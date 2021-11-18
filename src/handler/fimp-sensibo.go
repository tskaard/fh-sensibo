package handler

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/futurehomeno/fimpgo/edgeapp"
	"github.com/futurehomeno/fimpgo/utils"

	"github.com/futurehomeno/fimpgo"
	scribble "github.com/nanobox-io/golang-scribble"
	log "github.com/sirupsen/logrus"

	"github.com/tskaard/sensibo/model"
	"github.com/tskaard/sensibo/sensibo-api"
)

// FimpSensiboHandler structure
type FimpSensiboHandler struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	api          *sensibo.Sensibo
	db           *scribble.Driver
	state        model.State
	ticker       *time.Ticker
	configs      model.Configs
	appLifecycle *edgeapp.Lifecycle
	env          string
}

// NewFimpSensiboHandler construct new handler
func NewFimpSensiboHandler(transport *fimpgo.MqttTransport, stateFile string, appLifecycle *edgeapp.Lifecycle) *FimpSensiboHandler {
	fc := &FimpSensiboHandler{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: transport, appLifecycle: appLifecycle}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)

	hubInfo, err := utils.NewHubUtils().GetHubInfo()
	if err == nil && hubInfo != nil {
		fc.env = hubInfo.Environment
	} else {
		fc.env = utils.EnvProd
	}

	fc.api = sensibo.NewSensibo("")
	fc.db, _ = scribble.New(stateFile, nil)
	fc.state = model.State{}
	return fc
}

// Start start handler
func (fc *FimpSensiboHandler) Start(pollTimeSec int) error {
	if err := fc.db.Read("data", "state", &fc.state); err != nil {
		log.Error("Error loading state from file: ", err)
		fc.state.Connected = false
		log.Info("setting state connected to false")
		if err = fc.db.Write("data", "state", fc.state); err != nil {
			log.Error("Did not manage to write to file: ", err)
		}
	}
	fc.api.Key = fc.state.APIkey
	var errr error
	go func(msgChan fimpgo.MessageCh) {
		for {
			select {
			case newMsg := <-msgChan:
				fc.routeFimpMessage(newMsg)
			}
		}
	}(fc.inboundMsgCh)
	// Setting up ticker to poll information from cloud
	fc.ticker = time.NewTicker(time.Second * time.Duration(pollTimeSec))
	go func() {
		for range fc.ticker.C {
			// Check if app is connected
			if fc.state.Connected {
				log.Debug("")
				log.Debug("------------------TICKER START-------------------")

				pods, err := fc.api.GetPodsV1(fc.api.Key) // Update on  all ticks
				log.Debug("oldtemp after getpods: ", fc.state.Pods[0].OldTemp)
				if err != nil {
					log.Error("Can't get pods in tick")
				}

				oldPods := fc.state.Pods
				fc.state.Pods = pods

				for i, p := range fc.state.Pods {
					pod := p
					log.Debug("")
					log.Debug("pod.ID: ", pod.ID)

					if pod.MainMeasurementSensor.IsMainSensor {
						log.Debug("Sensor: external")
						temp := pod.MainMeasurementSensor.Measurements.Temperature
						humid := pod.MainMeasurementSensor.Measurements.Humidity
						motion := pod.MainMeasurementSensor.Measurements.Motion

						if oldPods[i].ExtOldTemp != temp {
							log.Debug("ext sensor temperature: ", temp)
							log.Debug("extoldTemp, ", oldPods[i].ExtOldTemp)
							fc.sendTemperatureMsg(pod.ID, temp, nil, 1)
						}
						fc.state.Pods[i].ExtOldTemp = temp

						if oldPods[i].ExtOldHumid != float64(humid) {
							log.Debug("ext sensor humidity: ", humid)
							log.Debug("extoldHumid, ", oldPods[i].ExtOldHumid)
							fc.sendHumidityMsg(pod.ID, float64(humid), nil, 1)
						}
						fc.state.Pods[i].ExtOldHumid = float64(humid)

						if oldPods[i].ExtOldMotion != motion {
							log.Debug("ext sensor motion: ", motion)
							log.Debug("extoldMotion, ", oldPods[i].ExtOldMotion)
							fc.SendMotionMsg(pod.ID, motion, nil)
						}
						fc.state.Pods[i].ExtOldMotion = motion

					}
					log.Debug("Sensor: internal")

					measurements, err := fc.api.GetMeasurements(pod.ID, fc.api.Key)
					if err != nil {
						log.Error("Cannot get measurements from internal device")
						continue
					}
					temp := measurements[0].Temperature
					if oldPods[i].OldTemp != temp {
						log.Debug("temp, ", temp)
						log.Debug("oldTemp, ", oldPods[i].OldTemp)
						fc.sendTemperatureMsg(pod.ID, temp, nil, 0)
					}
					fc.state.Pods[i].OldTemp = temp

					humid := measurements[0].Humidity
					if oldPods[i].OldHumid != humid {
						log.Debug("humid, ", humid)
						log.Debug("oldHumid, ", oldPods[i].OldHumid)
						fc.sendHumidityMsg(pod.ID, humid, nil, 0)
					}
					fc.state.Pods[i].OldHumid = humid

					states, err := fc.api.GetAcStates(pod.ID, fc.api.Key)
					if err != nil {
						log.Error("Faild to get current acState: ", err)
						continue
					}

					if len(states) > 0 {
						state := states[0].AcState
						if oldPods[i].OldState != state {
							log.Debug("state, ", state)
							log.Debug("oldState, ", oldPods[i].OldState)
							fc.sendAcState(pod.ID, state, nil, 0)
						}
						fc.state.Pods[i].OldState = state
					} else {
						log.Error("GetAcStates return empty results.")
					}
				}
			} else {
				log.Debug("------- NOT CONNECTED -------")
				// Do nothing
			}
		}
	}()
	return errr
}

func (fc *FimpSensiboHandler) routeFimpMessage(newMsg *fimpgo.Message) {
	// configs := model.Configs{}
	log.Debug("New fimp msg")
	if fc.state.IsConfigured() {
		fc.appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)
		fc.appLifecycle.SetConfigState(edgeapp.ConfigStateConfigured)
	}

	switch newMsg.Payload.Type {

	case "cmd.auth.set_tokens":
		err := newMsg.Payload.GetObjectValue(&fc.state)
		fc.appLifecycle.SetAuthState(edgeapp.AuthStateInProgress)
		if err != nil {
			log.Error("Wrong payload type , expected Object")
			fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
			return
		}

		fc.systemConnect(newMsg)
		fc.getAuthStatus(newMsg)

		fc.configs.SaveToFile()
		fc.state.SaveToFile()

	case "cmd.system.disconnect":
		fc.systemDisconnect(newMsg)
		fc.state.SaveToFile()

	case "cmd.auth.logout":
		fc.systemDisconnect(newMsg)
		fc.state.SaveToFile()

	case "cmd.app.uninstall":
		fc.systemDisconnect(newMsg)
		fc.state.SaveToFile()

	case "cmd.config.extended_set":
		err := newMsg.Payload.GetObjectValue(&fc.state)
		if err != nil {
			log.Debug("Can't get object value")
		}
		fanMode := fc.state.FanMode
		mode := fc.state.Mode
		fc.state.SaveToFile()

		for i := 0; i < len(fc.state.Pods); i++ {
			podID := fc.state.Pods[i].ID
			log.Debug(podID)
			adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:dev/rn:sensibo/ad:1/sv:fan_ctrl/ad:" + podID)
			msg := fimpgo.NewMessage("cmd.mode.set", "fan_ctrl", fimpgo.VTypeString, fanMode, nil, nil, newMsg.Payload)
			fc.mqt.Publish(adr, msg)

			adr, _ = fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + podID)
			msg = fimpgo.NewMessage("cmd.mode.set", "thermostat", fimpgo.VTypeString, mode, nil, nil, newMsg.Payload)
			fc.mqt.Publish(adr, msg)
		}
		configReport := model.ConfigReport{
			OpStatus: "ok",
			AppState: *fc.appLifecycle.GetAllStates(),
		}
		msg := fimpgo.NewMessage("evt.app.config_report", model.ServiceName, fimpgo.VTypeObject, configReport, nil, nil, newMsg.Payload)
		msg.Source = "sensibo"
		if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
			log.Debug("cant respond to wanted topic")
		}

	case "cmd.system.get_connect_params":
		fc.systemGetConnectionParameter(newMsg)

	case "cmd.system.connect":
		fc.systemConnect(newMsg)
		fc.state.SaveToFile()

	case "cmd.system.sync":
		var val model.ButtonActionResponse
		if fc.systemSync(newMsg) {
			val = model.ButtonActionResponse{
				Operation:       "cmd.system.sync",
				OperationStatus: "ok",
				Next:            "reload",
				ErrorCode:       "",
				ErrorText:       "",
			}
			log.Info("All devices synced")
		} else {
			val = model.ButtonActionResponse{
				Operation:       "cmd.system.sync",
				OperationStatus: "err",
				Next:            "reload",
				ErrorCode:       "",
				ErrorText:       "Sensibo is not connected",
			}
		}
		msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
		if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
			log.Error("Could not respond to wanted request")
		}

	case "cmd.setpoint.set":
		fc.setpointSet(newMsg)

	case "cmd.setpoint.get_report":
		fc.setpointGetReport(newMsg)

	case "cmd.mode.set":
		fc.modeSet(newMsg)

	case "cmd.mode.get_report":
		fc.modeGetReport(newMsg)

	case "cmd.thing.get_inclusion_report":
		fc.sendSingleInclusionReport(newMsg)
	//case "cmd.state.get_report":

	case "cmd.app.get_manifest":

		mode, err := newMsg.Payload.GetStringValue()
		if err != nil {
			log.Error("Incorrect request format ")
			return
		}
		manifest := edgeapp.NewManifest()
		err = manifest.LoadFromFile(filepath.Join(fc.configs.GetDefaultDir(), "app-manifest.json"))
		if err != nil {
			log.Error("Failed to load manifest file .Error :", err.Error())
			return
		}
		if mode == "manifest_state" {
			manifest.AppState = *fc.appLifecycle.GetAllStates()
			fc.configs.ConnectionState = string(fc.appLifecycle.ConnectionState())
			fc.configs.Errors = fc.appLifecycle.LastError()
			manifest.ConfigState = fc.state
		}

		if errConf := manifest.GetAppConfig("errors"); errConf != nil {
			if fc.configs.Errors == "" {
				errConf.Hidden = true
			} else {
				errConf.Hidden = false
			}
		}
		fanConfig := manifest.GetAppConfig("fan_ctrl")
		modeConfig := manifest.GetAppConfig("mode")
		fanConfig.Val.Default = fc.state.FanMode
		modeConfig.Val.Default = fc.state.Mode

		fanCtrlBlock := manifest.GetUIBlock("fan_ctrl")
		modeBlock := manifest.GetUIBlock("mode")
		syncButton := manifest.GetButton("sync")

		if fc.appLifecycle.ConnectionState() == edgeapp.ConnStateConnected {
			fanCtrlBlock.Hidden = false
			modeBlock.Hidden = false
			syncButton.Hidden = false
		} else {
			fanCtrlBlock.Hidden = true
			modeBlock.Hidden = true
			syncButton.Hidden = true
		}

		if fc.env == utils.EnvBeta {
			manifest.Auth.AuthEndpoint = "https://partners-beta.futurehome.io/api/edge/proxy/custom/auth-code"
			manifest.Auth.RedirectURL = "https://app-static-beta.futurehome.io/playground_oauth_callback"
		} else {
			manifest.Auth.AuthEndpoint = "https://partners.futurehome.io/api/edge/proxy/custom/auth-code"
			manifest.Auth.RedirectURL = "https://app-static.futurehome.io/playground_oauth_callback"
		}

		msg := fimpgo.NewMessage("evt.app.manifest_report", "sensibo", fimpgo.VTypeObject, manifest, nil, nil, newMsg.Payload)
		msg.Source = "sensibo"
		adr := &fimpgo.Address{MsgType: "rsp", ResourceType: "cloud", ResourceName: "remote-client", ResourceAddress: "smarthome-app"}
		if fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
			fc.mqt.Publish(adr, msg)
		}

	case "cmd.sensor.get_report":
		log.Debug("cmd.sensor.get_report")
		if !(newMsg.Payload.Service == "sensor_temp" || newMsg.Payload.Service == "sensor_humid") {
			log.Error("sensor.get_report - Wrong service")
			break
		}
		log.Debug("Getting measurements")
		address := strings.Replace(newMsg.Addr.ServiceAddress, "_0", "", 1)
		address = strings.Replace(address, "_1", "", 1)

		// address := newMsg.Addr.ServiceAddress
		measurements, err := fc.api.GetMeasurements(address, fc.api.Key)
		if err != nil {
			log.Error("Cannot get measurements from device")
			break
		}
		fc.state.Pods, err = fc.api.GetPodsV1(fc.api.Key)
		if err != nil {
			log.Error("Cannot get pods")
			break
		}
		var podNr int
		for i, p := range fc.state.Pods {
			pod := p
			if pod.ID == address {
				podNr = i
			}
		}

		switch newMsg.Payload.Service {
		case "sensor_temp":
			log.Debug("Getting temperature")
			temp := measurements[0].Temperature
			fc.sendTemperatureMsg(address, temp, newMsg.Payload, 0)
			if fc.state.Pods[podNr].MainMeasurementSensor.UID != "" {
				extTemp := fc.state.Pods[podNr].MainMeasurementSensor.Measurements.Temperature
				fc.sendTemperatureMsg(address, extTemp, newMsg.Payload, 1)
			}

		case "sensor_humid":
			log.Debug("Getting humidity")
			humid := measurements[0].Humidity
			fc.sendHumidityMsg(address, humid, newMsg.Payload, 0)
			if fc.state.Pods[podNr].MainMeasurementSensor.UID != "" {
				extHumid := fc.state.Pods[podNr].MainMeasurementSensor.Measurements.Humidity
				fc.sendHumidityMsg(address, float64(extHumid), newMsg.Payload, 1)
			}

		case "sensor_presence":
			log.Debug("Getting presence")
			extMotion := fc.state.Pods[podNr].MainMeasurementSensor.Measurements.Motion
			fc.SendMotionMsg(address, extMotion, newMsg.Payload)
		}

	case "cmd.thing.delete":
		// remove device from network
		val, err := newMsg.Payload.GetStrMapValue()
		if err != nil {
			log.Error("Wrong msg format")
			return
		}
		deviceID := val["address"]
		exists := 9999
		for i := 0; i < len(fc.state.Pods); i++ {
			podID := fc.state.Pods[i].ID
			if deviceID == podID {
				exists = i
			}
		}
		if exists != 9999 {
			fc.sendExclusionReport(deviceID, newMsg.Payload)
		}

	}
	fc.state.SaveToFile()
	fc.configs.SaveToFile()
}
