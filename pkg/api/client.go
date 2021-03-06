package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	url    string
	domain string
	key    string
}

func NewClient(url, domain, key string) (c *Client) {
	return &Client{
		url:    url,
		domain: domain,
		key:    key,
	}
}

func (c *Client) Do() (err error) {
	data := Data{
		Domain: c.domain,
		Key:    c.key,
	}

	dm, err := json.Marshal(data)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", c.url, bytes.NewBuffer(dm))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %v: %s", resp.StatusCode, resp.Status)
	}
	return
}
