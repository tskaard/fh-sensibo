package model

type State struct {
	Connected bool     `json:"connected"`
	APIkey    string   `json:"api_key"`
	Devices   []Device `json:"devices"`
}
