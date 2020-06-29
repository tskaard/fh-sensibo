package handler

import (
	"fmt"
	"math"
	"strconv"

	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

func checkSupportedFanMode(pod sensibo.Pod, mode string) bool {
	supported := false
	fanModes := getSupportedFanModes(pod)
	for _, m := range fanModes {
		if m == mode {
			supported = true
		}
	}
	return supported
}

func getSupportedFanModes(pod sensibo.Pod) []string {
	fimpModes := []string{"medium", "auto", "auto_low", "auto_high", "auto_mid", "low", "high", "mid", "humid_circulation", "up_down", "left_right", "quiet"}
	// sensibo "quiet", "low", "medium", "medium_high", "high", "auto"
	var fanModes []string
	for acMode, acValue := range pod.RemoteCapabilities.Modes {
		if acMode == "fan" {
			if acMapValue, ok := acValue.(map[string]interface{}); ok {
				for funcKey, funcVal := range acMapValue {
					if funcKey == "fanLevels" {
						if sModes, ok := funcVal.([]interface{}); ok {
							for _, sMode := range sModes {
								for _, fMode := range fimpModes {
									if sMode == fMode {
										fanModes = append(fanModes, fMode)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return fanModes
}

func getSupportedModes(pod sensibo.Pod) []string {
	var modes []string
	for acMode := range pod.RemoteCapabilities.Modes {
		modes = append(modes, acMode)
	}
	return modes
}

func getSupportedSetpoints(pod sensibo.Pod) []string {
	var modes []string
	for acMode, acValue := range pod.RemoteCapabilities.Modes {
		if acMapValue, ok := acValue.(map[string]interface{}); ok {
			for funcKey, funcValue := range acMapValue {
				if funcKey == "temperatures" {
					if temperatures, ok := funcValue.(map[string]interface{}); ok {
						for degKey, degValue := range temperatures {
							if degKey == "C" {
								if tempValue, ok := degValue.(map[string]interface{}); ok {
									for valueKey, data := range tempValue {
										if valueKey == "values" {
											if data != nil {
												modes = append(modes, acMode)
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
	return modes
}

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
