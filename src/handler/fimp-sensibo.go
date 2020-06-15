package handler

import (
	"time"

	"github.com/futurehomeno/fimpgo/edgeapp"

	"github.com/futurehomeno/fimpgo"
	scribble "github.com/nanobox-io/golang-scribble"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo-golang"
	"github.com/tskaard/sensibo/model"
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
	appLifecycle edgeapp.Lifecycle
}

// NewFimpSensiboHandler construct new handler
func NewFimpSensiboHandler(transport *fimpgo.MqttTransport, stateFile string) *FimpSensiboHandler {
	fc := &FimpSensiboHandler{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: transport}
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
					measurements, err := fc.api.GetMeasurements(pod.ID)
					if err != nil {
						log.Error("Cannot get measurements from device")
						break
					}
					temp := measurements[0].Temperature
					fc.sendTemperatureMsg(pod.ID, temp, nil)
					humid := measurements[0].Humidity
					fc.sendHumidityMsg(pod.ID, humid, nil)

					states, err := fc.api.GetAcStates(pod.ID)
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
		fc.systemConnect(newMsg)
		fc.systemGetConnectionParameter(newMsg)
		fc.systemSync(newMsg)
		fc.getAuthStatus(newMsg)

		fc.configs.SaveToFile()
		fc.state.SaveToFile()

	case "cmd.system.disconnect":
		fc.systemDisconnect(newMsg)
		fc.state.SaveToFile()

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
		mode, err := newMsg.Payload.GetStringValue()
		log.Debug(fc.configs)
		if err != nil {
			log.Error("Incorrect request format ")
			return
		}
		manifest := edgeapp.NewManifest()
		// err = manifest.LoadFromFile(filepath.Join(fc.configs.GetDefaultDir(), "app-manifest.json"))
		err = manifest.LoadFromFile("testdata/defaults/app-manifest.json")
		if err != nil {
			log.Error("Failed to load manifest file .Error :", err.Error())
			return
		}
		if mode == "manifest_state" {
			manifest.AppState = *fc.appLifecycle.GetAllStates()
			// manifest.AppState.Auth = string(fc.appLifecycle.AuthState())
			fc.configs.ConnectionState = string(fc.appLifecycle.ConnectionState())
			fc.configs.Errors = fc.appLifecycle.GetAllStates().LastErrorText
			confstate := make(map[string]interface{})
			// manifest.ConfigState = make(map[string]interface{})
			confstate["configured_at"] = fc.state.ConfiguredAt
			confstate["configured_by"] = fc.state.ConfiguredBy
			confstate["connected"] = fc.state.Connected
			confstate["api_key"] = fc.state.APIkey
			manifest.ConfigState = confstate
		}

		if errConf := manifest.GetAppConfig("errors"); errConf != nil {
			if fc.configs.Errors == "" {
				errConf.Hidden = true
			} else {
				errConf.Hidden = false
			}
		}
		connectButton := manifest.GetButton("connect")
		disconnectButton := manifest.GetButton("disconnect")
		if connectButton != nil && disconnectButton != nil {
			if fc.appLifecycle.ConnectionState() == edgeapp.ConnStateConnected {
				connectButton.Hidden = true
				disconnectButton.Hidden = false
			} else {
				connectButton.Hidden = false
				disconnectButton.Hidden = true
			}
		}
		if syncButton := manifest.GetButton("sync"); syncButton != nil {
			if fc.appLifecycle.ConnectionState() == edgeapp.ConnStateConnected {
				syncButton.Hidden = false
			} else {
				syncButton.Hidden = true
			}
		}
		connBlock := manifest.GetUIBlock("connect")
		settingsBlock := manifest.GetUIBlock("settings")
		if connBlock != nil && settingsBlock != nil {
			if fc.state.Pods != nil {
				connBlock.Hidden = false
				settingsBlock.Hidden = false
			} else {
				connBlock.Hidden = true
				settingsBlock.Hidden = true
			}
		}
		msg := fimpgo.NewMessage("evt.app.manifest_report", "sensibo", fimpgo.VTypeObject, manifest, nil, nil, newMsg.Payload)
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
		measurements, err := fc.api.GetMeasurements(address)
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
	}
}
