package handler

import (
	"strconv"

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
func (fc *FimpSensiboHandler) Start() error {
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
	return errr
}

func (fc *FimpSensiboHandler) routeFimpMessage(newMsg *fimpgo.Message) {
	log.Debug("New fimp msg")
	switch newMsg.Payload.Type {
	case "cmd.system.disconnect":
		log.Debug("cmd.system.disconnect")
		fc.SystemDisconnect(newMsg)

	case "cmd.system.get_connect_params":
		log.Debug("cmd.system.get_connect_params")
		// request api_key
		val := map[string]string{
			"security_key": "api_key",
			"address":      "localhost",
			"id":           "sensibo",
		}
		msg := fimpgo.NewStrMapMessage("evt.system.connect_params_report", "sensibo", val, nil, nil, newMsg.Payload)
		adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "sensibo", ResourceAddress: "1"}
		fc.mqt.Publish(&adr, msg)
		log.Debug("Connect params message sent")

	case "cmd.system.connect":
		// Do stuff to connect
		log.Debug("cmd.system.connect")

		val, err := newMsg.Payload.GetStrMapValue()
		if err != nil {
			log.Error("Wrong payload type , expected StrMap")
			break
		}
		if val["security_key"] == "" {
			log.Error("Did not get a security_key")
			break
		}
		fc.api.Key = val["security_key"]
		fc.state.APIkey = val["security_key"]
		// log.Debug(fc.api.Key)
		pods, err := fc.api.GetPods()
		if err != nil {
			log.Error("Cannot get pods information from Sensibo - ", err)
		}
		for _, pod := range pods {
			log.Debug(pod.ID)
			fc.SendInclusionReport(pod.ID, newMsg.Payload)
			fc.state.Devices = append(fc.state.Devices, model.Device{ID: pod.ID})
		}
		fc.state.Connected = true
		if err := fc.db.Write("data", "state", fc.state); err != nil {
			log.Error("Did not manage to write to file: ", err)
		}
		log.Debug("System connected")

	case "cmd.system.sync":
		log.Debug("cmd.system.sync")
		if !fc.state.Connected || fc.state.APIkey == "" {
			log.Error("Ad is not connected, not able to sync")
			break
		}
		for _, device := range fc.state.Devices {
			log.Debug(device.ID)
			fc.SendInclusionReport(device.ID, newMsg.Payload)
		}
		log.Debug("System synced")

	case "cmd.setpoint.set":
		if !(newMsg.Payload.Service == "thermostat") {
			log.Error("cmd.setpoint.set - Wrong service")
			break
		}
		address := newMsg.Addr.ServiceAddress

		value, err := newMsg.Payload.GetStrMapValue()
		if err != nil {
			log.Error("Could not get map of strings from thermostat setpoint set message", err)
			break
		}
		tempSetpointFloat, err := strconv.ParseFloat(value["temp"], 64)
		if err != nil {
			log.Error("Not able to convert string to float: ", err)
			break
		}
		// Sensibo only supports int between 16 and 30 deg
		tempSetpoint := int(tempSetpointFloat)
		if tempSetpoint <= 16 {
			tempSetpoint = 16
		}
		if tempSetpoint >= 30 {
			tempSetpoint = 30
		}
		acStates, err := fc.api.GetAcStates(address)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			break
		}
		newAcState := acStates[0].AcState
		newAcState.On = true
		newAcState.Mode = value["type"]
		newAcState.TargetTemperature = tempSetpoint
		if value["unit"] != "" {
			newAcState.TemperatureUnit = value["unit"]
		}
		acStateLog, err := fc.api.ReplaceState(address, newAcState)
		if err != nil {
			log.Error("Faild setting new AC state: ", err)
			break
		}
		log.Debug(acStateLog)
		// returning setpoint event
		value["temp"] = strconv.Itoa(tempSetpoint)
		msg := fimpgo.NewStrMapMessage("evt.setpoint.report", "thermostat", value, nil, nil, newMsg.Payload)
		adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + address)
		fc.mqt.Publish(adr, msg)
		log.Info("Thermostat setpoint changed")

	case "cmd.setpoint.get_report":
		if !(newMsg.Payload.Service == "thermostat") {
			log.Error("cmd.setpoint.get_report - Wrong service")
			break
		}
		address := newMsg.Addr.ServiceAddress
		states, err := fc.api.GetAcStates(address)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			break
		}
		state := states[0]
		val := make(map[string]string)
		val["temp"] = strconv.Itoa(state.AcState.TargetTemperature)
		val["type"] = state.AcState.Mode
		if state.AcState.TemperatureUnit != "" {
			val["unit"] = state.AcState.TemperatureUnit
		}
		msg := fimpgo.NewStrMapMessage("evt.setpoint.report", "thermostat", val, nil, nil, newMsg.Payload)
		adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + address)
		fc.mqt.Publish(adr, msg)

	case "cmd.mode.set":
		// Get current ac state and change the mode before sending it back
		if newMsg.Payload.Service == "thermostat" {
			address := newMsg.Addr.ServiceAddress
			newMode, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Could not get mode from thermostat mode set message", err)
				break
			}
			// Checking if we have a supported mode, and converting if needed
			if newMode == "auto_changeover" {
				newMode = "auto"
			} else if newMode == "dry_air" {
				newMode = "dry"
			}
			if !(newMode == "cool" || newMode == "heat" || newMode == "fan" || newMode == "auto" || newMode == "dry") {
				log.Error("Not supported thermostat mode : ", newMode)
				break
			}

			acStates, err := fc.api.GetAcStates(address)
			if err != nil {
				log.Error("Faild to get current acState: ", err)
				break
			}
			newAcState := acStates[0].AcState
			newAcState.Mode = newMode
			newAcState.On = true
			acStateLog, err := fc.api.ReplaceState(address, newAcState)
			if err != nil {
				log.Error("Faild setting new AC state: ", err)
				break
			}
			log.Debug(acStateLog)
			msg := fimpgo.NewStringMessage("evt.mode.report", "thermostat", newMode, nil, nil, newMsg.Payload)
			adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + address)
			fc.mqt.Publish(adr, msg)
		} else if newMsg.Payload.Service == "fan_ctrl" {
			address := newMsg.Addr.ServiceAddress
			newFanMode, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Could not get fan mode from fan_ctrl mode set message", err)
				break
			}
			if !(newFanMode == "quiet" || newFanMode == "low" || newFanMode == "medium" || newFanMode == "high" || newFanMode == "auto") {
				log.Error("Not supported fan mode : ", newFanMode)
				break
			}
			acStates, err := fc.api.GetAcStates(address)
			if err != nil {
				log.Error("Faild to get current acState: ", err)
				break
			}
			newAcState := acStates[0].AcState
			newAcState.FanLevel = newFanMode
			newAcState.On = true
			acStateLog, err := fc.api.ReplaceState(address, newAcState)
			if err != nil {
				log.Error("Faild setting new AC state: ", err)
				break
			}
			log.Debug(acStateLog)
			msg := fimpgo.NewStringMessage("evt.mode.report", "fan_ctrl", newFanMode, nil, nil, newMsg.Payload)
			adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:fan_ctrl/ad:" + address)
			fc.mqt.Publish(adr, msg)

		} else {
			log.Error("cmd.mode.set - Wrong service")
			break
		}

	case "cmd.mode.get_report":
		if newMsg.Payload.Service == "thermostat" {
			address := newMsg.Addr.ServiceAddress
			states, err := fc.api.GetAcStates(address)
			if err != nil {
				log.Error("Faild to get current acState: ", err)
				break
			}
			state := states[0].AcState
			mode := state.Mode
			msg := fimpgo.NewStringMessage("evt.setpoint.report", "thermostat", mode, nil, nil, newMsg.Payload)
			adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + address)
			fc.mqt.Publish(adr, msg)

		} else if newMsg.Payload.Service == "fan_ctrl" {
			address := newMsg.Addr.ServiceAddress
			states, err := fc.api.GetAcStates(address)
			if err != nil {
				log.Error("Faild to get current acState: ", err)
				break
			}
			state := states[0].AcState
			fanMode := state.FanLevel
			msg := fimpgo.NewStringMessage("evt.setpoint.report", "fan_ctrl", fanMode, nil, nil, newMsg.Payload)
			adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:fan_ctrl/ad:" + address)
			fc.mqt.Publish(adr, msg)

		} else {
			log.Error("cmd.setpoint.get_report - Wrong service")
			break
		}

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
			props := make(map[string]string)
			props["unit"] = "C"
			msg := fimpgo.NewMessage("evt.sensor.report", "sensor_temp", "float", temp, props, nil, newMsg.Payload)
			adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:sensor_temp/ad:" + address)
			fc.mqt.Publish(adr, msg)
			log.Info("Temperature message sent")
		case "sensor_humid":
			log.Debug("Getting humidity")
			humid := measurements[0].Humidity
			props := make(map[string]string)
			props["unit"] = "%"
			msg := fimpgo.NewMessage("evt.sensor.report", "sensor_humid", "float", humid, props, nil, newMsg.Payload)
			adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:sensor_humid/ad:" + address)
			fc.mqt.Publish(adr, msg)
			log.Info("Humidity message sent")
		}
	}
}
