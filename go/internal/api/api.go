package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/anomalyco/tuiflux/internal/models"
)

type Client struct {
	httpCli  *http.Client
	baseURL  string
	apiKey   string
	verifySSL bool
}

func New(serverURL, apiKey string, verifySSL bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !verifySSL,
		},
	}
	return &Client{
		httpCli:   &http.Client{Transport: tr},
		baseURL:   serverURL,
		apiKey:    apiKey,
		verifySSL: verifySSL,
	}
}

func (c *Client) do(method, path string, body, out interface{}) error {
	url := c.baseURL + path
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("X-Auth-Token", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %s", resp.Status)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

type entriesResponse struct {
	Entries []models.Entry `json:"entries"`
	Total   int            `json:"total"`
}

func (c *Client) GetFeeds() ([]models.Feed, error) {
	var feeds []models.Feed
	if err := c.do("GET", "/v1/feeds", nil, &feeds); err != nil {
		return nil, err
	}

	feedMap := make(map[int]*models.Feed)
	for i := range feeds {
		feeds[i].UnreadCount = 0
		feedMap[feeds[i].ID] = &feeds[i]
	}

	offset := 0
	limit := 1000
	for {
		var resp entriesResponse
		path := fmt.Sprintf("/v1/entries?status=unread&limit=%d&offset=%d", limit, offset)
		if err := c.do("GET", path, nil, &resp); err != nil {
			return nil, err
		}
		for _, e := range resp.Entries {
			if f, ok := feedMap[e.FeedID]; ok {
				f.UnreadCount++
			}
		}
		offset += limit
		if offset >= resp.Total || len(resp.Entries) == 0 {
			break
		}
	}

	var result []models.Feed
	for _, f := range feeds {
		if f.UnreadCount > 0 {
			result = append(result, f)
		}
	}
	return result, nil
}

func (c *Client) GetEntries(feedID int, status string, offset, limit int) ([]models.Entry, error) {
	path := fmt.Sprintf("/v1/feeds/%d/entries?status=%s&limit=%d&offset=%d",
		feedID, status, limit, offset)
	var resp entriesResponse
	if err := c.do("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Entries, nil
}

type updateStatusBody struct {
	EntryIDs []int  `json:"entry_ids"`
	Status   string `json:"status"`
}

func (c *Client) UpdateEntriesStatus(entryIDs []int, status string) error {
	return c.do("PUT", "/v1/entries", &updateStatusBody{
		EntryIDs: entryIDs,
		Status:   status,
	}, nil)
}

func (c *Client) ToggleStarred(entryID int) error {
	return c.do("PUT", fmt.Sprintf("/v1/entries/%d/bookmark", entryID), nil, nil)
}

type countersResponse struct {
	Unreads map[string]int `json:"unreads"`
}

func (c *Client) GetCounters() (map[string]int, error) {
	var resp countersResponse
	if err := c.do("GET", "/v1/feeds/counters", nil, &resp); err != nil {
		return c.getCountersFallback()
	}
	if resp.Unreads != nil {
		return resp.Unreads, nil
	}
	return c.getCountersFallback()
}

func (c *Client) getCountersFallback() (map[string]int, error) {
	counters := make(map[string]int)
	offset := 0
	limit := 1000
	for {
		var resp entriesResponse
		path := fmt.Sprintf("/v1/entries?status=unread&limit=%d&offset=%d", limit, offset)
		if err := c.do("GET", path, nil, &resp); err != nil {
			return nil, err
		}
		for _, e := range resp.Entries {
			fid := strconv.Itoa(e.FeedID)
			counters[fid]++
		}
		offset += limit
		if offset >= resp.Total || len(resp.Entries) == 0 {
			break
		}
	}
	return counters, nil
}

func (c *Client) GetReadEntriesCount() (int, error) {
	var resp entriesResponse
	if err := c.do("GET", "/v1/entries?status=read&limit=1", nil, &resp); err != nil {
		return 0, err
	}
	return resp.Total, nil
}

func (c *Client) FlushHistory() error {
	return c.do("PUT", "/v1/flush-history", nil, nil)
}
