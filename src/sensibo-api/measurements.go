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

func (s *Sensibo) GetMeasurements(deviceID string, apiKey string) ([]Measurement, error) {
	req, err := http.NewRequest("GET", s.resourceUrl(fmt.Sprintf("pods/%v/measurements", deviceID), url.Values{}), nil)
	if err != nil {
		log.Error(fmt.Errorf("Can't get measurements - ", err))
		return nil, err
	}
	req.Header.Set("Authorization", os.ExpandEnv(fmt.Sprintf("%s%s", "Bearer ", apiKey)))
	resp, err := http.DefaultClient.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result MeasurementResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, errors.New("Cannot get pods")
	}

	return result.Result, nil
}
