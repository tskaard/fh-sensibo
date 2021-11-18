package sensibo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"
)

type RemoteCapabilities struct {
	Modes map[string]interface{} `json:"modes"`
}

type Room struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type MainMeasurementSensor struct {
	UID          string `json:"uid"`
	QrID         string `json:"qrId"`
	ConfigGroup  string `json:"configGroup"`
	IsAlive      bool   `json:"isAlive"`
	IsMainSensor bool   `json:"isMainSensor"`
	Measurements struct {
		Temperature    float64     `json:"temperature"`
		Humidity       int         `json:"humidity"`
		FeelsLike      float64     `json:"feelsLike"`
		Rssi           int         `json:"rssi"`
		Motion         bool        `json:"motion"`
		RoomIsOccupied interface{} `json:"roomIsOccupied"`
		BatteryVoltage int         `json:"batteryVoltage"`
	} `json:"measurements"`
}

type Pod struct {
	ID                 string             `json:"id"`
	Room               Room               `json:"room"`
	MacAddress         string             `json:"macAddress"`
	ProductModel       string             `json:"productModel"`
	RemoteCapabilities RemoteCapabilities `json:"remoteCapabilities"`
	OldTemp            float64            `json:"oldTemp"`
	OldHumid           float64            `json:"oldHumid"`
	OldState           AcState            `json:"oldState"`
}

type PodV1 struct {
	ID                    string                `json:"podUid"`
	Room                  Room                  `json:"room"`
	MacAddress            string                `json:"macAddress"`
	ProductModel          string                `json:"productModel"`
	MainMeasurementSensor MainMeasurementSensor `json:"mainMeasurementsSensor"`
	RemoteCapabilities    RemoteCapabilities    `json:"remoteCapabilities"`
	OldTemp               float64               `json:"oldTemp"`
	OldHumid              float64               `json:"oldHumid"`
	OldState              AcState               `json:"oldState"`
	OldMotion             bool                  `json:"oldMotion"`
	ExtOldTemp            float64               `json:"externalOldTemp"`
	ExtOldHumid           float64               `json:"externalOldHumid"`
	ExtOldMotion          bool                  `json:"externalOldMotion"`
}

type PodResult struct {
	Status string `json:"status"`
	Result []Pod  `json:"result"`
}

type PodResultV1 struct {
	Pods []PodV1 `json:"pods"`
}

// GetPods gets information on all pods on account
func (s *Sensibo) GetPods(apiKey string) ([]Pod, error) {
	values := url.Values{
		"fields": []string{"id,room,productModel,macAddress,remoteCapabilities"},
	}
	req, err := http.NewRequest("GET", s.resourceUrl("users/me/pods", values), nil)
	if err != nil {
		log.Error(fmt.Errorf("Can't send pods request - ", err))
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", os.ExpandEnv(fmt.Sprintf("%s%s", "Bearer ", apiKey)))
	log.Debug(req)
	resp, err := http.DefaultClient.Do(req)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result PodResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, errors.New("Cannot get pods")
	}

	defer resp.Body.Close()
	return result.Result, nil
}

// GetPodsV1 gets information on all pods on account with V1 API, which includes MainMeasurementSensor/external motion/temp/humid sensor
func (s *Sensibo) GetPodsV1(apiKey string) ([]PodV1, error) {
	req, err := http.NewRequest("GET", s.resourceUrlV1("users/me/pods", nil), nil)
	if err != nil {
		log.Error(fmt.Errorf("Can't send pods v1 request - ", err))
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", os.ExpandEnv(fmt.Sprintf("%s%s", "Bearer ", apiKey)))
	resp, err := http.DefaultClient.Do(req)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result PodResultV1
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	if result.Pods == nil {
		return nil, errors.New("Cannot get pods")
	}

	defer resp.Body.Close()
	return result.Pods, nil
}
