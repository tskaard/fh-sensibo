package handler

import (
	"path/filepath"
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
}

// NewFimpSensiboHandler construct new handler
func NewFimpSensiboHandler(transport *fimpgo.MqttTransport, stateFile string, appLifecycle *edgeapp.Lifecycle) *FimpSensiboHandler {
	fc := &FimpSensiboHandler{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: transport, appLifecycle: appLifecycle}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)
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
			// ADD timer from config
			if fc.state.Connected {
				for _, pod := range fc.state.Pods {
					measurements, err := fc.api.GetMeasurements(pod.ID, fc.api.Key)
					if err != nil {
						log.Error("Cannot get measurements from device")
						break
					}
					temp := measurements[0].Temperature
					fc.sendTemperatureMsg(pod.ID, temp, nil)
					humid := measurements[0].Humidity
					fc.sendHumidityMsg(pod.ID, humid, nil)

					states, err := fc.api.GetAcStates(pod.ID, fc.api.Key)
					if err != nil {
						log.Error("Faild to get current acState: ", err)
						break
					}
					state := states[0].AcState
					fc.sendAcState(pod.ID, state, nil)
				}
				// Get measurements and acState from all units
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
		fc.systemSync(newMsg)

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
		var env string
		hubInfo, err := utils.NewHubUtils().GetHubInfo()
		if err == nil && hubInfo != nil {
			env = hubInfo.Environment
		} else {
			env = utils.EnvBeta
		}

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

		if fanCtrlBlock != nil && modeBlock != nil {
			if fc.state.Pods != nil {
				fanCtrlBlock.Hidden = false
				modeBlock.Hidden = false
			} else {
				fanCtrlBlock.Hidden = true
				modeBlock.Hidden = true
			}
		}

		if syncButton := manifest.GetButton("sync"); syncButton != nil {
			if fc.appLifecycle.ConnectionState() == edgeapp.ConnStateConnected {
				syncButton.Hidden = false
			} else {
				syncButton.Hidden = false
			}
		}

		if env == utils.EnvBeta {
			manifest.Auth.AuthEndpoint = "https://partners-beta.futurehome.io/api/edge/proxy/custom/auth-code"
			manifest.Auth.RedirectURL = "https://app-static-beta.futurehome.io/playground_oauth_callback"
		} else {
			manifest.Auth.AuthEndpoint = "https://app-static.futurehome.io/playground_oauth_callback"
			manifest.Auth.RedirectURL = "https://partners.futurehome.io/api/edge/proxy/custom/auth-code"
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
		address := newMsg.Addr.ServiceAddress
		measurements, err := fc.api.GetMeasurements(address, fc.api.Key)
		if err != nil {
			log.Error("Cannot get measurements from device")
			break
		}
		switch newMsg.Payload.Service {
		case "sensor_temp":
			log.Debug("Getting temperature")
			temp := measurements[0].Temperature
			fc.sendTemperatureMsg(address, temp, newMsg.Payload)
		case "sensor_humid":
			log.Debug("Getting humidity")
			humid := measurements[0].Humidity
			fc.sendHumidityMsg(address, humid, newMsg.Payload)
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
