package handler

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	scribble "github.com/nanobox-io/golang-scribble"
	log "github.com/sirupsen/logrus"
	"github.com/tskaard/sensibo/model"
	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

// FimpSensiboHandler structure
type FimpSensiboHandler struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	api          *sensibo.Sensibo
	db           *scribble.Driver
	state        model.State
	ticker       *time.Ticker
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
	log.Debug("New fimp msg")
	switch newMsg.Payload.Type {
	case "cmd.system.disconnect":
		fc.systemDisconnect(newMsg)

	case "cmd.system.get_connect_params":
		fc.systemGetConnectionParameter(newMsg)

	case "cmd.system.connect":
		fc.systemConnect(newMsg)

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

	//case "cmd.state.get_report":

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
