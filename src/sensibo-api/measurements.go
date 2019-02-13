package sensibo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type Time struct {
	SecondsAgo int    `json:"secondAgo"`
	Time       string `json:"time"`
}

type Measurement struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
	Time        Time    `json:"time"`
}

type MeasurementResult struct {
	Status string        `json:"status"`
	Result []Measurement `json:"result"`
}

func (s *Sensibo) GetMeasurements(deviceID string) ([]Measurement, error) {
	data, err := s.get(fmt.Sprintf("pods/%v/measurements", deviceID), url.Values{})
	if err != nil {
		return nil, err
	}

	var result MeasurementResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, errors.New("Cannot get pods")
	}

	return result.Result, nil
}
