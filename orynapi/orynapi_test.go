package orynapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPlayers(t *testing.T) {
	response := PlayersResponse{
		Count: 2,
		Players: []Player{
			{Username: "Player1", UUID: "uuid1", Ping: 50, Server: "server1", Address: "127.0.0.1"},
			{Username: "Player2", UUID: "uuid2", Ping: 100, Server: "server1", Address: "127.0.0.1"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/players" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.FetchPlayers(context.Background())
	if err != nil {
		t.Fatalf("FetchPlayers() error = %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Count = %d, want %d", result.Count, 2)
	}
	if len(result.Players) != 2 {
		t.Errorf("Players length = %d, want %d", len(result.Players), 2)
	}
	if result.Players[0].Username != "Player1" {
		t.Errorf("Players[0].Username = %q, want %q", result.Players[0].Username, "Player1")
	}
}

func TestFetchPlayersServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.FetchPlayers(context.Background())
	if err == nil {
		t.Error("FetchPlayers() should return error for server error")
	}
}

func TestFetchPlayersInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.FetchPlayers(context.Background())
	if err == nil {
		t.Error("FetchPlayers() should return error for invalid JSON")
	}
}

func TestFetchPlayersContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PlayersResponse{})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient(server.URL)
	_, err := client.FetchPlayers(ctx)
	if err == nil {
		t.Error("FetchPlayers() should return error for cancelled context")
	}
}
