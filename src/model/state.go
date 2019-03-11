package model

import (
	sensibo "github.com/tskaard/sensibo-golang"
)

type State struct {
	Connected bool          `json:"connected"`
	APIkey    string        `json:"api_key"`
	Pods      []sensibo.Pod `json:"pods"`
}
