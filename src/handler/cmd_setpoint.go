package handler

import (
	"strings"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

func (fc *FimpSensiboHandler) setpointGetReport(oldMsg *fimpgo.Message) {
	if !(oldMsg.Payload.Service == "thermostat") {
		log.Error("cmd.setpoint.get_report - Wrong service")
		return
	}
	address := strings.Replace(oldMsg.Addr.ServiceAddress, "_0", "", 1)
	address = strings.Replace(address, "_1", "", 1)
	states, err := fc.api.GetAcStates(address, fc.api.Key)
	if err != nil {
		log.Error("Faild to get current acState: ", err)
		return
	}
	if len(states) < 1 {
		log.Error("States is empty.")
		return
	}

	state := states[0].AcState
	var devicetype string
	for _, pod := range fc.state.Pods {
		if pod.ID == address {
			devicetype = "air"
		}
	}
	if devicetype == "air" {
		fc.sendSetpointMsg(address, state, oldMsg.Payload, 0)
	} else {
		fc.sendSetpointMsg(address, state, oldMsg.Payload, -1)
	}
}

func (fc *FimpSensiboHandler) setpointSet(oldMsg *fimpgo.Message) {
	if !(oldMsg.Payload.Service == "thermostat") {
		log.Error("cmd.setpoint.set - Wrong service")
		return
	}
	address := strings.Replace(oldMsg.Addr.ServiceAddress, "_0", "", 1)
	address = strings.Replace(address, "_1", "", 1)
	value, err := oldMsg.Payload.GetStrMapValue()
	if err != nil {
		log.Error("Could not get map of strings from thermostat setpoint set message", err)
		return
	}
	// var pod sensibo.Pod
	var pod sensibo.PodV1
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
	setpointMode := value["type"]
	tempSetpoint := getSupportedSetpoint(pod, value)
	acStates, err := fc.api.GetAcStates(pod.ID, fc.api.Key)
	if err != nil {
		log.Error("Faild to get current acState: ", err)
		return
	}
	fc.state.Mode = setpointMode
	newAcState := acStates[0].AcState
	newAcState.On = true
	newAcState.Mode = setpointMode
	newAcState.TargetTemperature = tempSetpoint
	if value["unit"] != "" {
		newAcState.TemperatureUnit = value["unit"]
	}
	acStateLog, err := fc.api.ReplaceState(pod.ID, newAcState, fc.api.Key)
	if err != nil {
		log.Error("Faild setting new AC state: ", err)
		// not break bur return something?
		return
	}
	log.Debug(acStateLog)
	var devicetype string
	if pod.MainMeasurementSensor.UID != "" {
		devicetype = "air"
	}

	if devicetype == "air" {
		fc.sendSetpointMsg(pod.ID, newAcState, oldMsg.Payload, 0)
	} else {
		fc.sendSetpointMsg(pod.ID, newAcState, oldMsg.Payload, -1)
	}
}
