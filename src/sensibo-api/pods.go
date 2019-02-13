package sensibo

import (
	"encoding/json"
	"errors"
	"net/url"
)

type Room struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type Pod struct {
	ID   string `json:"id"`
	Room Room   `json:"room"`
}

type PodResult struct {
	Status string `json:"status"`
	Result []Pod  `json:"result"`
}

func (s *Sensibo) GetPods() ([]Pod, error) {
	values := url.Values{
		"fields": []string{"id,room"},
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
