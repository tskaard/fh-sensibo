package handler

import (
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

func (fc *FimpSensiboHandler) systemDisconnect(msg *fimpgo.Message) {
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

func (fc *FimpSensiboHandler) sendExclusionReport(addr string, oldMsg *fimpgo.FimpMessage) {
	exReport := fimptype.ThingExclusionReport{
		Address: addr,
	}
	msg := fimpgo.NewMessage("evt.thing.exclusion_report", "sensibo", "object", exReport, nil, nil, oldMsg)
	adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "sensibo", ResourceAddress: "1"}
	fc.mqt.Publish(&adr, msg)
	log.Debug("Exclusion report sent")
}
