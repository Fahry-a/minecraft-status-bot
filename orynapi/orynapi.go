package orynapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Player struct {
	Username string `json:"username"`
	UUID     string `json:"uuid"`
	Ping     int    `json:"ping"`
	Server   string `json:"server"`
	Address  string `json:"address"`
}

type PlayersResponse struct {
	Count     int      `json:"count"`
	UpdatedAt int64    `json:"updatedAt"`
	Players   []Player `json:"players"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) FetchPlayers() (*PlayersResponse, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/players", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch players: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result PlayersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
