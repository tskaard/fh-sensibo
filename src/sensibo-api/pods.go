package sensibo

import (
	"encoding/json"
	"errors"
	"net/url"
)

type Temperatures struct {
	C struct {
		IsNative bool  `json:"isNative"`
		Values   []int `json:"values"`
	} `json:"C"`
}

type Modes struct {
	Dry struct {
		Swing        []string     `json:"swing"`
		Temperatures Temperatures `json:"temperatures"`
		FanLevels    []string     `json:"fanLevels"`
	} `json:"dry"`
	Auto struct {
		Swing        []string     `json:"swing"`
		Temperatures Temperatures `json:"temperatures"`
		FanLevels    []string     `json:"fanLevels"`
	} `json:"auto"`
	Heat struct {
		Swing        []string     `json:"swing"`
		Temperatures Temperatures `json:"temperatures"`
		FanLevels    []string     `json:"fanLevels"`
	} `json:"heat"`
	Fan struct {
		Swing        []string     `json:"swing"`
		Temperatures Temperatures `json:"temperatures"`
		FanLevels    []string     `json:"fanLevels"`
	} `json:"fan"`
	Cool struct {
		Swing        []string     `json:"swing"`
		Temperatures Temperatures `json:"temperatures"`
		FanLevels    []string     `json:"fanLevels"`
	} `json:"cool"`
}

type RemoteCapabilities struct {
	Modes Modes `json:"modes"`
}

type Room struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type Pod struct {
	ID                 string             `json:"id"`
	Room               Room               `json:"room"`
	MacAddress         string             `json:"macAddress"`
	ProductModel       string             `json:"productModel"`
	RemoteCapabilities RemoteCapabilities `json:"remoteCapabilities"`
}

type PodResult struct {
	Status string `json:"status"`
	Result []Pod  `json:"result"`
}

// GetPods gets information on all pods on account
func (s *Sensibo) GetPods() ([]Pod, error) {
	values := url.Values{
		"fields": []string{"id,room,productModel,macAddress,remoteCapabilities"},
	}
	data, err := s.get("users/me/pods", values)
	if err != nil {
		return nil, err
	}

	var result PodResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, errors.New("Cannot get pods")
	}

	return result.Result, nil
}
