package handler

import (
	"fmt"
	"math"
	"strconv"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo-golang"
)

func checkSupportedSetpointMode(pod sensibo.Pod, mode string) bool {
	supported := false
	modes := getModes(pod)
	for _, m := range modes {
		if m == mode {
			supported = true
		}
	}
	return supported
}

func getModes(p sensibo.Pod) []string {
	var modes []string
	for acMode := range p.RemoteCapabilities.Modes {
		modes = append(modes, acMode)
	}
	return modes
}

func getSupportedSetpoint(pod sensibo.Pod, value map[string]string) int {
	var temperature int
	tempF, err := strconv.ParseFloat(value["temp"], 64)
	if err != nil {
		fmt.Println("Not able to convert string to float: ", err)
		return 0
	}
	temp := int(math.Round(tempF))
	temps := getSupportedTempRange(pod, value)
	if temp <= temps[0] {
		temperature = temps[0]
	} else if temp >= temps[len(temps)-1] {
		temperature = temps[len(temps)-1]
	} else {
		for _, t := range temps {
			if temp == t {
				temperature = temp
				break
			}
		}
	}
	return temperature
}

func getSupportedTempRange(pod sensibo.Pod, value map[string]string) []int {
	mode := value["type"]
	var tempRange []int
	for acMode, acValue := range pod.RemoteCapabilities.Modes {
		if acMode == mode {
			if acMapValue, ok := acValue.(map[string]interface{}); ok {
				for funcKey, funcValue := range acMapValue {
					if funcKey == "temperatures" {
						if temperatures, ok := funcValue.(map[string]interface{}); ok {
							for degKey, degValue := range temperatures {
								if degKey == "C" {
									if tempValue, ok := degValue.(map[string]interface{}); ok {
										for valueKey, data := range tempValue {
											if valueKey == "values" {
												if temps, ok := data.([]interface{}); ok {
													for _, v := range temps {
														tempRange = append(tempRange, int(math.Round(v.(float64))))
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return tempRange
}

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
