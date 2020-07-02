package handler

import (
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

func (fc *FimpSensiboHandler) sendExclusionReport(addr string, oldMsg *fimpgo.FimpMessage) {
	exReport := fimptype.ThingExclusionReport{
		Address: addr,
	}
	msg := fimpgo.NewMessage("evt.thing.exclusion_report", "sensibo", "object", exReport, nil, nil, oldMsg)
	msg.Source = "sensibo"
	adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "sensibo", ResourceAddress: "1"}
	fc.mqt.Publish(&adr, msg)
	log.Debug("Exclusion report sent")
}
