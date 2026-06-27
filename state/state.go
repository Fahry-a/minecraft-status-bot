package state

import (
	"encoding/json"
	"os"
	"time"
)

type State struct {
	Maintenance     bool      `json:"maintenance"`
	MaintenanceAt   time.Time `json:"maintenanceAt,omitempty"`
	LastOnline      time.Time `json:"lastOnline,omitempty"`
	LastOffline     time.Time `json:"lastOffline,omitempty"`
	FirstSeen       time.Time `json:"firstSeen,omitempty"`
	LastSave        time.Time `json:"lastSave"`
	TotalRestarts   int       `json:"totalRestarts"`
}

func Load(path string) *State {
	data, err := os.ReadFile(path)
	if err != nil {
		return &State{FirstSeen: time.Now()}
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return &State{FirstSeen: time.Now()}
	}

	s.TotalRestarts++
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
