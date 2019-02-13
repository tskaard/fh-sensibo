package sensibo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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

func (s *Sensibo) GetAcStates(deviceID string) ([]AcStateLog, error) {
	data, err := s.get(fmt.Sprintf("pods/%v/acStates", deviceID), url.Values{})
	if err != nil {
		return nil, err
	}

	var result AcStatesResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, errors.New("Cannot get ac states")
	}

	return result.Result, nil
}

func (s *Sensibo) ReplaceState(deviceID string, state AcState) (*AcStateLog, error) {
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
	data, err := s.post(fmt.Sprintf("pods/%v/acStates", deviceID), currentStateData)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

func (s *Sensibo) UpdateState(deviceID string, property string, state AcStateChange) (*AcStateLog, error) {
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
	data, err := s.patch(fmt.Sprintf("pods/%v/acStates/%v", deviceID, property), currentStateData)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}
