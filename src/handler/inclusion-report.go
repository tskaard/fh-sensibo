package handler

import (
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

func buildInterface(iType string, msgType string, valueType string, version string) fimptype.Interface {
	inter := fimptype.Interface{
		Type:      iType,
		MsgType:   msgType,
		ValueType: valueType,
		Version:   version,
	}
	return inter
}

func buildSensorService(addr string, service string, supUnits []string, alias string) fimptype.Service {
	cmdSensorGetReport := buildInterface("in", "cmd.sensor.get_report", "null", "1")
	evtSensorReport := buildInterface("out", "evt.sensor.report", "float", "1")
	sensorInterfaces := []fimptype.Interface{}
	sensorInterfaces = append(sensorInterfaces, cmdSensorGetReport, evtSensorReport)

	props := make(map[string]interface{})
	props["sup_units"] = supUnits
	sensorService := fimptype.Service{
		Address:    "/rt:dev/rn:sensibo/ad:1/sv:" + service + "/ad:" + addr,
		Name:       service,
		Groups:     []string{"ch_0"},
		Alias:      alias,
		Enabled:    true,
		Props:      props,
		Interfaces: sensorInterfaces,
	}
	return sensorService
}

func buildThermostatService(pod sensibo.Pod) fimptype.Service {
	evtSetpointReport := buildInterface("out", "evt.setpoint.report", "str_map", "1")
	cmdSetpointSet := buildInterface("in", "cmd.setpoint.set", "str_map", "1")
	cmdSetpointGetReport := buildInterface("in", "cmd.setpoint.get_report", "string", "1")
	cmdModeSet := buildInterface("in", "cmd.mode.set", "string", "1")
	cmdModeGetReport := buildInterface("in", "cmd.mode.get_report", "null", "1")
	evtModeReport := buildInterface("out", "evt.mode.report", "string", "1")
	evtStateReport := buildInterface("out", "evt.state.report", "string", "1")
	cmdStateGetReport := buildInterface("in", "cmd.state.get_report", "null", "1")

	thermostatInterfaces := []fimptype.Interface{}
	thermostatInterfaces = append(
		thermostatInterfaces, evtSetpointReport, cmdSetpointSet, cmdSetpointGetReport,
		cmdModeSet, cmdModeGetReport, evtModeReport, evtStateReport, cmdStateGetReport)

	props := make(map[string]interface{})
	props["sup_modes"] = getSupportedModes(pod)
	props["sup_setpoints"] = getSupportedSetpoints(pod)
	props["sup_units"] = []string{"C", "F"}
	thermostatService := fimptype.Service{
		Address:    "/rt:dev/rn:sensibo/ad:1/sv:thermostat/ad:" + pod.ID,
		Name:       "thermostat",
		Groups:     []string{"ch_0"},
		Alias:      "thermostat",
		Enabled:    true,
		Props:      props,
		Interfaces: thermostatInterfaces,
	}
	return thermostatService
}

func buildFanCtrlService(pod sensibo.Pod) fimptype.Service {
	cmdModeSet := buildInterface("in", "cmd.mode.set", "string", "1")
	cmdModeGetReport := buildInterface("in", "cmd.mode.get_report", "null", "1")
	evtModeReport := buildInterface("out", "evt.mode.report", "string", "1")

	fanCtrlInterfaces := []fimptype.Interface{}
	fanCtrlInterfaces = append(fanCtrlInterfaces, cmdModeSet, cmdModeGetReport, evtModeReport)

	props := make(map[string]interface{})
	//props["sup_modes"] = []string{"quiet", "low", "medium", "high", "auto"}
	props["sup_modes"] = getSupportedFanModes(pod)
	fanCtrlService := fimptype.Service{
		Address:    "/rt:dev/rn:sensibo/ad:1/sv:fan_ctrl/ad:" + pod.ID,
		Name:       "fan_ctrl",
		Groups:     []string{"ch_0"},
		Alias:      "fan_ctrl",
		Enabled:    true,
		Props:      props,
		Interfaces: fanCtrlInterfaces,
	}
	return fanCtrlService
}

func (fc *FimpSensiboHandler) sendInclusionReport(pod sensibo.Pod, oldMsg *fimpgo.FimpMessage) {

	tempSensorService := buildSensorService(pod.ID, "sensor_temp", []string{"C"}, "temperature")
	humidSensorService := buildSensorService(pod.ID, "sensor_humid", []string{"%"}, "humidity")

	thermostatService := buildThermostatService(pod)
	fanCtrlService := buildFanCtrlService(pod)

	services := []fimptype.Service{}
	services = append(services, tempSensorService, humidSensorService, thermostatService, fanCtrlService)
	incReort := fimptype.ThingInclusionReport{
		Address:        pod.ID,
		HwVersion:      pod.ProductModel,
		CommTechnology: "http",
		ProductName:    "Sensibo Sky",
		Groups:         []string{"ch_0"},
		Services:       services,
		Alias:          pod.Room.Name,
		ProductId:      pod.ProductModel,
		DeviceId:       pod.MacAddress,
	}

	msg := fimpgo.NewMessage("evt.thing.inclusion_report", "sensibo", "object", incReort, nil, nil, nil)
	adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "sensibo", ResourceAddress: "1"}
	fc.mqt.Publish(&adr, msg)
	log.Debug("Inclusion report sent")
}

func (fc *FimpSensiboHandler) sendSingleInclusionReport(oldMsg *fimpgo.Message) {
	deviceId, err := oldMsg.Payload.GetStringValue()
	if err != nil {
		log.Debug("could not get string value from payload")
	}
	for _, pod := range fc.state.Pods {
		log.Debug(pod.ID)
		if pod.ID == deviceId {
			fc.sendInclusionReport(pod, oldMsg.Payload)
		}
	}
}
