package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/futurehomeno/fimpgo/edgeapp"
	"github.com/futurehomeno/fimpgo/utils"
	log "github.com/sirupsen/logrus"
	sensibo "github.com/tskaard/sensibo/sensibo-api"
)

type State struct {
	path         string
	WorkDir      string        `json:"-"`
	ConfiguredAt string        `json:"configured_at"`
	ConfiguredBy string        `json:"configured_by"`
	Connected    bool          `json:"connected"`
	APIkey       string        `json:"access_token"`
	Pods         []sensibo.Pod `json:"pods"`
	FanMode      string        `json:"fan_ctrl"`
	Mode         string        `json:"mode"`
}

func NewStates(workDir string) *State {
	state := &State{WorkDir: workDir}
	state.InitFiles()
	return state
}

func (st *State) InitFiles() error {
	st.path = filepath.Join(st.WorkDir, "data", "state.json")
	if !utils.FileExists(st.path) {
		log.Info("State file doesn't exist.Loading default state")
		defaultStateFile := filepath.Join(st.WorkDir, "defaults", "state.json")
		err := utils.CopyFile(defaultStateFile, st.path)
		if err != nil {
			fmt.Print(err)
			panic("Can't copy state file.")
		}
	}
	return nil
}

func (st *State) LoadFromFile() error {
	stateFileBody, err := ioutil.ReadFile(st.path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(stateFileBody, st)
	if err != nil {
		return err
	}
	return nil
}

func (st *State) SaveToFile() error {
	st.ConfiguredBy = "auto"
	st.ConfiguredAt = time.Now().Format(time.RFC3339)
	bpayload, err := json.Marshal(st)
	err = ioutil.WriteFile(st.path, bpayload, 0664)
	if err != nil {
		return err
	}
	return err
}

func (st *State) GetDataDir() string {
	return filepath.Join(st.WorkDir, "data")
}

func (st *State) GetDefaultDir() string {
	return filepath.Join(st.WorkDir, "defaults")
}

func (st *State) LoadDefaults() error {
	stateFile := filepath.Join(st.WorkDir, "data", "config.json")
	os.Remove(stateFile)
	log.Info("Config file doesn't exist.Loading default config")
	defaultConfigFile := filepath.Join(st.WorkDir, "defaults", "config.json")
	return utils.CopyFile(defaultConfigFile, stateFile)
}

func (st *State) IsConfigured() bool {
	if st.APIkey != "" {
		return true
	}
	return false
}

type PublicStates struct {
	ConnectionState string `json:"connection_state"`
	Errors          string `json:"errors"`
}

type StateReport struct {
	OpStatus string            `json:"op_status"`
	AppState edgeapp.AppStates `json:"app_state"`
}
