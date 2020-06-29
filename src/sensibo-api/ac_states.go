package sensibo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"
)

type AcState struct {
	On                bool   `json:"on"`
	FanLevel          string `json:"fanLevel"`
	TemperatureUnit   string `json:"temperatureUnit"`
	TargetTemperature int    `json:"targetTemperature"`
	Mode              string `json:"mode"`
	Swing             string `json:"swing"`
}

type AcStateLog struct {
	Status            string   `json:"status"`
	Reason            string   `json:"reason"`
	ID                string   `json:"id"`
	FailureReason     string   `json:"failureReason"`
	ChangedProperties []string `json:"changedProperties"`
	AcState           AcState  `json:"acState"`
}
type AcStateChange struct {
	NewValue string  `json:"newValue"`
	AcState  AcState `json:"currentAcState"`
}

type AcStatesResult struct {
	Status string       `json:"status"`
	Result []AcStateLog `json:"result"`
}

// GetAcStates gets the AC state from a pod
func (s *Sensibo) GetAcStates(deviceID string, apiKey string) ([]AcStateLog, error) {
	req, err := http.NewRequest("GET", s.resourceUrl(fmt.Sprintf("pods/%v/acStates", deviceID), url.Values{}), nil)
	if err != nil {
		log.Error(fmt.Errorf("Can't get ac states - ", err))
		return nil, err
	}
	req.Header.Set("Authorization", os.ExpandEnv(fmt.Sprintf("%s%s", "Bearer ", apiKey)))
	resp, err := http.DefaultClient.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result AcStatesResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, errors.New("Cannot get ac states")
	}

	return result.Result, nil
}

func (s *Sensibo) ReplaceState(deviceID string, state AcState, apiKey string) (*AcStateLog, error) {
	currentStateData, err := json.Marshal(struct {
		AcState AcState `json:"acState"`
	}{state})
	if err != nil {
		return nil, err
	}

	var result struct {
		Status string     `json:"status"`
		Result AcStateLog `json:"result"`
	}
	buffer := bytes.NewBuffer(currentStateData)
	req, err := http.NewRequest("POST", s.resourceUrl(fmt.Sprintf("pods/%v/acStates", deviceID), url.Values{}), buffer)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", os.ExpandEnv(fmt.Sprintf("%s%s", "Bearer ", apiKey)))
	resp, err := http.DefaultClient.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

func (s *Sensibo) UpdateState(deviceID string, property string, state AcStateChange, apiKey string) (*AcStateLog, error) {
	currentStateData, err := json.Marshal(struct {
		AcState AcStateChange `json:"currentAcState"`
	}{state})
	if err != nil {
		return nil, err
	}

	var result struct {
		Status string     `json:"status"`
		Result AcStateLog `json:"result"`
	}
	// data, err := s.patch(fmt.Sprintf("pods/%v/acStates/%v", deviceID, property), currentStateData)
	buffer := bytes.NewBuffer(currentStateData)
	req, err := http.NewRequest("POST", s.resourceUrl(fmt.Sprintf("pods/%g/acStates/%v", deviceID, property), url.Values{}), buffer)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", os.ExpandEnv(fmt.Sprintf("%s%s", "Bearer ", apiKey)))
	resp, err := http.DefaultClient.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}
