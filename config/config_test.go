package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	configContent := `{
		"token": "test-token",
		"serverIP": "mc.example.com",
		"serverPort": 25565,
		"channelID": "123456789",
		"orynApiUrl": "http://api.example.com",
		"updateInterval": 5000
	}`

	tmpFile, err := os.CreateTemp("", "config-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Token != "test-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "test-token")
	}
	if cfg.ServerIP != "mc.example.com" {
		t.Errorf("ServerIP = %q, want %q", cfg.ServerIP, "mc.example.com")
	}
	if cfg.ServerPort != 25565 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 25565)
	}
	if cfg.UpdateInterval != 5000 {
		t.Errorf("UpdateInterval = %d, want %d", cfg.UpdateInterval, 5000)
	}
}

func TestLoadConfigDefaultInterval(t *testing.T) {
	configContent := `{
		"token": "test-token",
		"serverIP": "mc.example.com",
		"serverPort": 25565,
		"channelID": "123456789"
	}`

	tmpFile, err := os.CreateTemp("", "config-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.UpdateInterval != 10000 {
		t.Errorf("UpdateInterval = %d, want %d", cfg.UpdateInterval, 10000)
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	configContent := `{
		"token": "file-token",
		"serverIP": "mc.example.com",
		"serverPort": 25565,
		"channelID": "123456789"
	}`

	tmpFile, err := os.CreateTemp("", "config-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpFile.Close()

	os.Setenv("DISCORD_TOKEN", "env-token")
	defer os.Unsetenv("DISCORD_TOKEN")

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want %q (env should override file)", cfg.Token, "env-token")
	}
}

func TestValidateMissingToken(t *testing.T) {
	cfg := &Config{
		ServerIP:   "mc.example.com",
		ServerPort: 25565,
		ChannelID:  "123456789",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing token")
	}
}

func TestValidateMissingServerIP(t *testing.T) {
	cfg := &Config{
		Token:      "test-token",
		ServerPort: 25565,
		ChannelID:  "123456789",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing server IP")
	}
}

func TestValidateMissingChannelID(t *testing.T) {
	cfg := &Config{
		Token:      "test-token",
		ServerIP:   "mc.example.com",
		ServerPort: 25565,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing channel ID")
	}
}

func TestValidateInvalidPort(t *testing.T) {
	cfg := &Config{
		Token:      "test-token",
		ServerIP:   "mc.example.com",
		ServerPort: 99999,
		ChannelID:  "123456789",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for invalid port")
	}
}

func TestValidateValidConfig(t *testing.T) {
	cfg := &Config{
		Token:      "test-token",
		ServerIP:   "mc.example.com",
		ServerPort: 25565,
		ChannelID:  "123456789",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}
