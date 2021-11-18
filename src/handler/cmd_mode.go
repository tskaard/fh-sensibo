package handler

import (
	"strings"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

func (fc *FimpSensiboHandler) modeGetReport(oldMsg *fimpgo.Message) {
	if oldMsg.Payload.Service == "thermostat" {
		address := strings.Replace(oldMsg.Addr.ServiceAddress, "_0", "", 1)
		address = strings.Replace(address, "_1", "", 1)
		states, err := fc.api.GetAcStates(address, fc.api.Key)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		mode := states[0].AcState.Mode
		var devicetype string
		for _, pod := range fc.state.Pods {
			if pod.ID == address {
				devicetype = "air"
			}
		}
		if devicetype == "air" {
			fc.sendThermostatModeMsg(address, mode, oldMsg.Payload, 0)
		} else {
			fc.sendThermostatModeMsg(address, mode, oldMsg.Payload, -1)
		}

	} else if oldMsg.Payload.Service == "fan_ctrl" {
		address := strings.Replace(oldMsg.Addr.ServiceAddress, "_0", "", 1)
		address = strings.Replace(address, "_1", "", 1)
		states, err := fc.api.GetAcStates(address, fc.api.Key)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		fanMode := states[0].AcState.FanLevel
		var devicetype string
		for _, pod := range fc.state.Pods {
			if pod.ID == address {
				devicetype = "air"
			}
		}
		if devicetype == "air" {
			fc.sendFanCtrlMsg(address, fanMode, oldMsg.Payload, 0)
		} else {
			fc.sendFanCtrlMsg(address, fanMode, oldMsg.Payload, -1)
		}
	} else {
		log.Error("cmd.setpoint.get_report - Wrong service")
		return
	}

}

func (fc *FimpSensiboHandler) modeSet(oldMsg *fimpgo.Message) {
	newMode, err := oldMsg.Payload.GetStringValue()
	if err != nil {
		log.Error("Could not get mode from thermostat mode set message", err)
		return
	}
	log.Debug(newMode)
	if newMode == "off" {
		log.Debug("turn off sensibo device")
		fc.turnOff(oldMsg)
		// turn off sensibo device
	} else {
		fc.activeModeSet(oldMsg)
	}

}

func (fc *FimpSensiboHandler) turnOff(oldMsg *fimpgo.Message) {
	if oldMsg.Payload.Service == "thermostat" {
		address := strings.Replace(oldMsg.Addr.ServiceAddress, "_0", "", 1)
		address = strings.Replace(address, "_1", "", 1)
		var pod sensibo.PodV1
		for _, p := range fc.state.Pods {
			if p.ID == address {
				pod = p
				break
			}
		}
		acStates, err := fc.api.GetAcStates(pod.ID, fc.api.Key)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		newAcState := acStates[0].AcState
		newAcState.On = false
		acStateLog, err := fc.api.ReplaceState(pod.ID, newAcState, fc.api.Key)
		if err != nil {
			log.Error("Faild setting new AC state: ", err)
			return
		}
		log.Debug(acStateLog)
		var devicetype string
		for _, pod := range fc.state.Pods {
			if pod.ID == address {
				devicetype = "air"
			}
		}
		if devicetype == "air" {
			fc.sendThermostatModeMsg(pod.ID, "off", oldMsg.Payload, 0)
		} else {
			fc.sendThermostatModeMsg(pod.ID, "off", oldMsg.Payload, -1)
		}
	}
}

func (fc *FimpSensiboHandler) activeModeSet(oldMsg *fimpgo.Message) {
	if oldMsg.Payload.Service == "thermostat" {
		address := strings.Replace(oldMsg.Addr.ServiceAddress, "_0", "", 1)
		address = strings.Replace(address, "_1", "", 1)
		newMode, err := oldMsg.Payload.GetStringValue()
		if err != nil {
			log.Error("Could not get mode from thermostat mode set message", err)
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
		var devicetype string
		for _, pod := range fc.state.Pods {
			if pod.ID == address {
				devicetype = "air"
			}
		}
		if devicetype == "air" {
			fc.sendThermostatModeMsg(pod.ID, newMode, oldMsg.Payload, 0)
		} else {
			fc.sendThermostatModeMsg(pod.ID, newMode, oldMsg.Payload, -1)
		}

	} else if oldMsg.Payload.Service == "fan_ctrl" {
		address := strings.Replace(oldMsg.Addr.ServiceAddress, "_0", "", 1)
		address = strings.Replace(address, "_1", "", 1)
		newFanMode, err := oldMsg.Payload.GetStringValue()
		if err != nil {
			log.Error("Could not get fan mode from fan_ctrl mode set message", err)
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
		var devicetype string
		for _, pod := range fc.state.Pods {
			if pod.ID == address {
				devicetype = "air"
			}
		}
		if devicetype == "air" {
			fc.sendFanCtrlMsg(pod.ID, newFanMode, oldMsg.Payload, 0)
		} else {
			fc.sendFanCtrlMsg(pod.ID, newFanMode, oldMsg.Payload, -1)
		}
	} else {
		log.Error("cmd.mode.set - Wrong service")
		return
	}
}
