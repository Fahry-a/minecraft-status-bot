package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if v := os.Getenv("DISCORD_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("SERVER_IP"); v != "" {
		cfg.ServerIP = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.ServerPort = port
		}
	}
	if v := os.Getenv("CHANNEL_ID"); v != "" {
		cfg.ChannelID = v
	}
	if v := os.Getenv("ORYN_API_URL"); v != "" {
		cfg.OrynApiUrl = v
	}
	if v := os.Getenv("UPDATE_INTERVAL"); v != "" {
		if interval, err := strconv.Atoi(v); err == nil {
			cfg.UpdateInterval = interval
		}
	}

	if cfg.UpdateInterval == 0 {
		cfg.UpdateInterval = 10000
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("discord token is required (set DISCORD_TOKEN env or token in config)")
	}
	if c.ServerIP == "" {
		return fmt.Errorf("server IP is required (set SERVER_IP env or serverIP in config)")
	}
	if c.ChannelID == "" {
		return fmt.Errorf("channel ID is required (set CHANNEL_ID env or channelID in config)")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", c.ServerPort)
	}
	return nil
}
