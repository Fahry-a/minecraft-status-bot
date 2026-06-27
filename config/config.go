package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Token          string `json:"token"`
	ServerIP       string `json:"serverIP"`
	ServerPort     int    `json:"serverPort"`
	ChannelID      string `json:"channelID"`
	OrynApiUrl     string `json:"orynApiUrl"`
	UpdateInterval int    `json:"updateInterval"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.UpdateInterval == 0 {
		cfg.UpdateInterval = 10000
	}

	return &cfg, nil
}
