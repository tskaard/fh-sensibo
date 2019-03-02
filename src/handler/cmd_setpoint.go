package handler

import (
	"strconv"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

func (fc *FimpSensiboHandler) setpointGetReport(oldMsg *fimpgo.Message) {
	if !(oldMsg.Payload.Service == "thermostat") {
		log.Error("cmd.setpoint.get_report - Wrong service")
		return
	}
	address := oldMsg.Addr.ServiceAddress
	states, err := fc.api.GetAcStates(address)
	if err != nil {
		log.Error("Faild to get current acState: ", err)
		return
	}
	state := states[0].AcState
	fc.sendSetpointMsg(address, state, oldMsg.Payload)
}

func (fc *FimpSensiboHandler) setpointSet(oldMsg *fimpgo.Message) {
	if !(oldMsg.Payload.Service == "thermostat") {
		log.Error("cmd.setpoint.set - Wrong service")
		return
	}
	address := oldMsg.Addr.ServiceAddress
	value, err := oldMsg.Payload.GetStrMapValue()
	if err != nil {
		log.Error("Could not get map of strings from thermostat setpoint set message", err)
		return
	}
	tempSetpointFloat, err := strconv.ParseFloat(value["temp"], 64)
	if err != nil {
		log.Error("Not able to convert string to float: ", err)
		return
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
		return
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
		// not break bur return something?
		return
	}
	log.Debug(acStateLog)
	fc.sendSetpointMsg(address, newAcState, oldMsg.Payload)
}
