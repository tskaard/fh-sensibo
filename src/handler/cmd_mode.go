package handler

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

func (fc *FimpSensiboHandler) modeGetReport(oldMsg *fimpgo.Message) {
	if oldMsg.Payload.Service == "thermostat" {
		address := oldMsg.Addr.ServiceAddress
		states, err := fc.api.GetAcStates(address)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		mode := states[0].AcState.Mode
		fc.sendThermostatModeMsg(address, mode, oldMsg.Payload)

	} else if oldMsg.Payload.Service == "fan_ctrl" {
		address := oldMsg.Addr.ServiceAddress
		states, err := fc.api.GetAcStates(address)
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
		// Checking if we have a supported mode, and converting if needed
		if newMode == "auto_changeover" {
			newMode = "auto"
		} else if newMode == "dry_air" {
			newMode = "dry"
		}
		if !(newMode == "cool" || newMode == "heat" || newMode == "fan" || newMode == "auto" || newMode == "dry") {
			log.Error("Not supported thermostat mode : ", newMode)
			return
		}

		acStates, err := fc.api.GetAcStates(address)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		newAcState := acStates[0].AcState
		newAcState.Mode = newMode
		newAcState.On = true
		acStateLog, err := fc.api.ReplaceState(address, newAcState)
		if err != nil {
			log.Error("Faild setting new AC state: ", err)
			return
		}
		log.Debug(acStateLog)
		fc.sendThermostatModeMsg(address, newMode, oldMsg.Payload)

	} else if oldMsg.Payload.Service == "fan_ctrl" {
		address := oldMsg.Addr.ServiceAddress
		newFanMode, err := oldMsg.Payload.GetStringValue()
		if err != nil {
			log.Error("Could not get fan mode from fan_ctrl mode set message", err)
			return
		}
		if !(newFanMode == "quiet" || newFanMode == "low" || newFanMode == "medium" || newFanMode == "high" || newFanMode == "auto") {
			log.Error("Not supported fan mode : ", newFanMode)
			return
		}
		acStates, err := fc.api.GetAcStates(address)
		if err != nil {
			log.Error("Faild to get current acState: ", err)
			return
		}
		newAcState := acStates[0].AcState
		newAcState.FanLevel = newFanMode
		newAcState.On = true
		acStateLog, err := fc.api.ReplaceState(address, newAcState)
		if err != nil {
			log.Error("Faild setting new AC state: ", err)
			return
		}
		log.Debug(acStateLog)
		fc.sendFanCtrlMsg(address, newFanMode, oldMsg.Payload)
	} else {
		log.Error("cmd.mode.set - Wrong service")
		return
	}
}
