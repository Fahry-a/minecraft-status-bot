package state

import (
	"encoding/json"
	"os"
	"time"
)

type State struct {
	Maintenance bool      `json:"maintenance"`
	LastSave    time.Time `json:"lastSave"`
}

func Load(path string) *State {
	data, err := os.ReadFile(path)
	if err != nil {
		return &State{}
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return &State{}
	}
	return &s
}

func Save(path string, s *State) error {
	s.LastSave = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
