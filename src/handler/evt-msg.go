package handler

import (
	"strconv"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

func (fc *FimpSensiboHandler) sendTemperatureMsg(addr string, temp float64, oldMsg *fimpgo.FimpMessage) {
	props := make(map[string]string)
	props["unit"] = "C"
	msg := fimpgo.NewMessage("evt.sensor.report", "sensor_temp", "float", temp, props, nil, oldMsg)
	adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:sensor_temp/ad:" + addr)
	fc.mqt.Publish(adr, msg)
	log.Debug("Temperature message sent")
}

func (fc *FimpSensiboHandler) sendHumidityMsg(addr string, humid float64, oldMsg *fimpgo.FimpMessage) {
	props := make(map[string]string)
	props["unit"] = "%"
	msg := fimpgo.NewMessage("evt.sensor.report", "sensor_humid", "float", humid, props, nil, oldMsg)
	adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:sensor_humid/ad:" + addr)
	fc.mqt.Publish(adr, msg)
	log.Debug("Humidity message sent")
}

func (fc *FimpSensiboHandler) sendThermostatModeMsg(addr string, mode string, oldMsg *fimpgo.FimpMessage) {
	msg := fimpgo.NewStringMessage("evt.mode.report", "thermostat", mode, nil, nil, oldMsg)
	adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + addr)
	fc.mqt.Publish(adr, msg)
	log.Debug("Thermostat mode message sent")
}

func (fc *FimpSensiboHandler) sendFanCtrlMsg(addr string, fanMode string, oldMsg *fimpgo.FimpMessage) {
	msg := fimpgo.NewStringMessage("evt.mode.report", "fan_ctrl", fanMode, nil, nil, oldMsg)
	adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:fan_ctrl/ad:" + addr)
	fc.mqt.Publish(adr, msg)
	log.Debug("Fan ctrl mode message sent")
}

func (fc *FimpSensiboHandler) sendSetpointMsg(addr string, acState sensibo.AcState, oldMsg *fimpgo.FimpMessage) {
	val := make(map[string]string)
	val["temp"] = strconv.Itoa(acState.TargetTemperature)
	val["type"] = acState.Mode
	if acState.TemperatureUnit != "" {
		val["unit"] = acState.TemperatureUnit
	}
	msg := fimpgo.NewStrMapMessage("evt.setpoint.report", "thermostat", val, nil, nil, oldMsg)
	adr, _ := fimpgo.NewAddressFromString("pt:j1/mt:evt/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + addr)
	fc.mqt.Publish(adr, msg)
	log.Debug("Thermostat setpoint message sent")
}

func (fc *FimpSensiboHandler) sendAcState(addr string, acState sensibo.AcState, oldMsg *fimpgo.FimpMessage) {
	if acState.Mode == "" {
		log.Error("AcState does not include Mode")
	} else {
		fc.state.Mode = acState.Mode
		mode := acState.Mode
		fc.sendThermostatModeMsg(addr, mode, oldMsg)
	}
	if acState.TargetTemperature == 0 {
		log.Error("Setpoint temperature is not included in acState")
	} else {
		fc.sendSetpointMsg(addr, acState, oldMsg)
	}
	if acState.FanLevel == "" {
		log.Error("Fan Level is not included in acState")
	} else {
		fc.state.FanMode = acState.FanLevel
		fanMode := acState.FanLevel
		fc.sendFanCtrlMsg(addr, fanMode, oldMsg)
	}
}
