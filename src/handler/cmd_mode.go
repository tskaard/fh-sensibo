package handler

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

func (fc *FimpSensiboHandler) modeGetReport(oldMsg *fimpgo.Message) {
	if oldMsg.Payload.Service == "thermostat" {
		address := oldMsg.Addr.ServiceAddress
		states, err := fc.api.GetAcStates(address, fc.api.Key)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		mode := states[0].AcState.Mode
		fc.sendThermostatModeMsg(address, mode, oldMsg.Payload)

	} else if oldMsg.Payload.Service == "fan_ctrl" {
		address := oldMsg.Addr.ServiceAddress
		states, err := fc.api.GetAcStates(address, fc.api.Key)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		fanMode := states[0].AcState.FanLevel
		fc.sendFanCtrlMsg(address, fanMode, oldMsg.Payload)
	} else {
		log.Error("cmd.setpoint.get_report - Wrong service")
		return
	}

}

func (fc *FimpSensiboHandler) modeSet(oldMsg *fimpgo.Message) {
	if oldMsg.Payload.Service == "thermostat" {
		address := oldMsg.Addr.ServiceAddress
		newMode, err := oldMsg.Payload.GetStringValue()
		if err != nil {
			log.Error("Could not get mode from thermostat mode set message", err)
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
		if !checkSupportedSetpointMode(pod, newMode) {
			log.Error("Setpoint mode is not supported")
			return
		}
		acStates, err := fc.api.GetAcStates(pod.ID, fc.api.Key)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		newAcState := acStates[0].AcState
		newAcState.Mode = newMode
		newAcState.On = true
		acStateLog, err := fc.api.ReplaceState(pod.ID, newAcState, fc.api.Key)
		if err != nil {
			log.Error("Faild setting new AC state: ", err)
			return
		}
		log.Debug(acStateLog)
		fc.sendThermostatModeMsg(pod.ID, newMode, oldMsg.Payload)

	} else if oldMsg.Payload.Service == "fan_ctrl" {
		address := oldMsg.Addr.ServiceAddress
		newFanMode, err := oldMsg.Payload.GetStringValue()
		if err != nil {
			log.Error("Could not get fan mode from fan_ctrl mode set message", err)
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
		if !checkSupportedFanMode(pod, newFanMode) {
			log.Error("Fan mode is not supported")
			return
		}
		if newFanMode == "mid" {
			newFanMode = "medium"
		}
		acStates, err := fc.api.GetAcStates(pod.ID, fc.api.Key)
		if err != nil {
			log.Error("Failed to get current acState: ", err)
			return
		}
		newAcState := acStates[0].AcState
		newAcState.FanLevel = newFanMode
		newAcState.On = true
		acStateLog, err := fc.api.ReplaceState(pod.ID, newAcState, fc.api.Key)
		if err != nil {
			log.Error("Faild setting new AC state: ", err)
			return
		}
		log.Debug(acStateLog)
		fc.sendFanCtrlMsg(pod.ID, newFanMode, oldMsg.Payload)
	} else {
		log.Error("cmd.mode.set - Wrong service")
		return
	}
}
