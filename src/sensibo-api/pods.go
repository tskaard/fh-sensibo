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

type PodResult struct {
	Status string `json:"status"`
	Result []Pod  `json:"result"`
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
