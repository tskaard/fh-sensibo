package handler

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo-golang"
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
	var pod sensibo.Pod
	for _, p := range fc.state.Pods {
		if p.ID == address {
			pod = p
			break
		}
	}
	if pod.ID == "" {
		log.Error("Address of pod is not stored in state")
		return
	}
	if !checkSupportedSetpointMode(pod, value["type"]) {
		log.Error("Setpoint mode is not supported")
		return
	}
	setpointMode := value["type"]
	tempSetpoint := getSupportedSetpoint(pod, value)
	acStates, err := fc.api.GetAcStates(pod.ID)
	if err != nil {
		log.Error("Faild to get current acState: ", err)
		return
	}
	newAcState := acStates[0].AcState
	newAcState.On = true
	newAcState.Mode = setpointMode
	newAcState.TargetTemperature = tempSetpoint
	if value["unit"] != "" {
		newAcState.TemperatureUnit = value["unit"]
	}
	acStateLog, err := fc.api.ReplaceState(pod.ID, newAcState)
	if err != nil {
		log.Error("Faild setting new AC state: ", err)
		// not break bur return something?
		return
	}
	log.Debug(acStateLog)
	fc.sendSetpointMsg(pod.ID, newAcState, oldMsg.Payload)
}
