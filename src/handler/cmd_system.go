package handler

import (
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/edgeapp"
	log "github.com/sirupsen/logrus"
)

func (fc *FimpSensiboHandler) systemSync(oldMsg *fimpgo.Message) {
	pods, err := fc.api.GetPods(fc.api.Key)
	if err != nil {
		log.Error("Can't get pods from Sensibo")
	}
	fc.state.Pods = pods
	log.Debug("cmd.system.sync")
	if !fc.state.Connected || fc.state.APIkey == "" {
		log.Error("Ad is not connected, not able to sync")
		return
	}
	fc.appLifecycle.SetConnectionState(edgeapp.ConnStateConnecting)
	for _, pod := range fc.state.Pods {
		log.Debug(pod.ID)
		fc.sendInclusionReport(pod, oldMsg.Payload)
	}
	fc.appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)
	log.Info("System synced")
}

func (fc *FimpSensiboHandler) getAuthStatus(oldMsg *fimpgo.Message) {
	log.Debug("cmd.auth.set_tokens")
	val := fc.appLifecycle.GetAllStates()

	msg := fimpgo.NewMessage("evt.auth.status_report", "sensibo", fimpgo.VTypeObject, val, nil, nil, oldMsg.Payload)
	msg.Source = "sensibo"
	if err := fc.mqt.RespondToRequest(oldMsg.Payload, msg); err != nil {
		log.Error("Could not respond to wanted request")
	}
}

func (fc *FimpSensiboHandler) systemDisconnect(msg *fimpgo.Message) {
	log.Debug("cmd.system.disconnect")
	if !fc.state.Connected {
		log.Error("Ad is not connected, no devices to exclude")
		val2 := map[string]interface{}{
			"errors":  nil,
			"success": false,
		}
		newMsg := fimpgo.NewMessage("evt.pd7.response", "vinculum", fimpgo.VTypeObject, val2, nil, nil, msg.Payload)
		if err := fc.mqt.RespondToRequest(msg.Payload, newMsg); err != nil {
			log.Error("Could not respond to wanted request")
		}
		return
	}
	for _, pod := range fc.state.Pods {
		fc.sendExclusionReport(pod.ID, msg.Payload)
	}
	fc.appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)
	fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
	fc.appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
	fc.state.Connected = false
	fc.state.APIkey = ""
	fc.state.Pods = nil
	if err := fc.db.Write("data", "state", fc.state); err != nil {
		log.Error("Did not manage to write to file: ", err)
	}
	val2 := map[string]interface{}{
		"errors":  nil,
		"success": true,
	}
	newMsg := fimpgo.NewMessage("evt.pd7.response", "vinculum", fimpgo.VTypeObject, val2, nil, nil, msg.Payload)
	newMsg.Source = "sensibo"
	if err := fc.mqt.RespondToRequest(msg.Payload, newMsg); err != nil {
		log.Error("Could not respond to wanted request")
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
		val["access_token"] = fc.state.APIkey
	} else {
		val["access_token"] = "api_key"
	}
	msg := fimpgo.NewStrMapMessage("evt.system.connect_params_report", "sensibo", val, nil, nil, nil)
	msg.Source = "sensibo"
	adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "sensibo", ResourceAddress: "1"}
	fc.mqt.Publish(&adr, msg)
	log.Debug("Connect params message sent")
}

func (fc *FimpSensiboHandler) systemConnect(oldMsg *fimpgo.Message) {
	log.Debug("cmd.system.connect")
	if fc.state.Connected {
		log.Error("App is already connected with system")
		fc.appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)
		return
	}
	if fc.state.APIkey == "" {
		log.Error("Did not get a security_key")
		fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
		return
	}

	fc.api.Key = fc.state.APIkey
	pods, err := fc.api.GetPods(fc.api.Key)
	if err != nil {
		log.Error("Cannot get pods information from Sensibo - ", err)
		fc.api.Key = ""
		fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
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

	if fc.state.APIkey != "" {
		fc.state.Connected = true
		fc.appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
	} else {
		fc.state.Connected = false
		fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
	}

	if err := fc.db.Write("data", "state", fc.state); err != nil {
		log.Error("Did not manage to write to file: ", err)
		return
	}
	log.Debug("System connected")
}
