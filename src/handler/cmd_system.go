package handler

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

func (fc *FimpSensiboHandler) systemSync(oldMsg *fimpgo.Message) {
	log.Debug("cmd.system.sync")
	if !fc.state.Connected || fc.state.APIkey == "" {
		log.Error("Ad is not connected, not able to sync")
		return
	}
	for _, pod := range fc.state.Pods {
		log.Debug(pod.ID)
		fc.sendInclusionReport(pod, oldMsg.Payload)
	}
	log.Info("System synced")
}

func (fc *FimpSensiboHandler) systemDisconnect(msg *fimpgo.Message) {
	log.Debug("cmd.system.disconnect")
	if !fc.state.Connected {
		log.Error("Ad is not connected, no devices to exclude")
		return
	}
	// TODO Change pod id yo use address from file
	for _, pod := range fc.state.Pods {
		fc.sendExclusionReport(pod.ID, msg.Payload)
	}
	fc.state.Connected = false
	fc.state.APIkey = ""
	fc.state.Pods = nil
	if err := fc.db.Write("data", "state", fc.state); err != nil {
		log.Error("Did not manage to write to file: ", err)
	}

}

func (fc *FimpSensiboHandler) systemGetConnectionParameter(oldMsg *fimpgo.Message) {
	log.Debug("cmd.system.get_connect_params")
	// request api_key
	val := map[string]string{
		"address": "localhost",
		"id":      "sensibo",
	}
	if fc.state.Connected {
		val["security_key"] = fc.state.APIkey
	} else {
		val["security_key"] = "api_key"
	}
	msg := fimpgo.NewStrMapMessage("evt.system.connect_params_report", "sensibo", val, nil, nil, oldMsg.Payload)
	adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "sensibo", ResourceAddress: "1"}
	fc.mqt.Publish(&adr, msg)
	log.Debug("Connect params message sent")
}

func (fc *FimpSensiboHandler) systemConnect(oldMsg *fimpgo.Message) {
	log.Debug("cmd.system.connect")
	if fc.state.Connected {
		log.Error("App is already connected with system")
		return
	}
	val, err := oldMsg.Payload.GetStrMapValue()
	if err != nil {
		log.Error("Wrong payload type , expected StrMap")
		return
	}
	if val["security_key"] == "" {
		log.Error("Did not get a security_key")
		return
	}

	fc.api.Key = val["security_key"]
	pods, err := fc.api.GetPods()
	if err != nil {
		log.Error("Cannot get pods information from Sensibo - ", err)
		fc.api.Key = ""
		return
	}

	for _, pod := range pods {
		log.Debug(pod.ID)
		log.Debug(pod.ProductModel)
		if pod.ProductModel != "skyv2" {
			return
		}
		fc.state.Pods = append(fc.state.Pods, pod)
		fc.sendInclusionReport(pod, oldMsg.Payload)
	}
	if fc.state.Pods != nil {
		fc.state.APIkey = val["security_key"]
		fc.state.Connected = true
	} else {
		fc.state.APIkey = ""
		fc.state.Connected = false
	}

	if err := fc.db.Write("data", "state", fc.state); err != nil {
		log.Error("Did not manage to write to file: ", err)
		return
	}
	log.Debug("System connected")
}
