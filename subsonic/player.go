package subsonic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) GetRandomSongs() ([]Song, error) {
	params := c.buildParams(map[string]string{
		"size":   fmt.Sprintf("%d", c.PageSize),
		"format": "json",
	})
	requestUrl := fmt.Sprintf("%s/rest/getRandomSongs?%s", c.BaseURL, params.Encode())
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var subsonicResp struct {
		SubsonicResponse struct {
			Status string `json:"status"`
			Error  struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			RandomSongs struct {
				Songs []Song `json:"song"`
			} `json:"randomSongs"`
		} `json:"subsonic-response"`
	}

	if err := json.Unmarshal(body, &subsonicResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if subsonicResp.SubsonicResponse.Status != "ok" {
		return nil, fmt.Errorf("subsonic error %d: %s",
			subsonicResp.SubsonicResponse.Error.Code,
			subsonicResp.SubsonicResponse.Error.Message)
	}

	return subsonicResp.SubsonicResponse.RandomSongs.Songs, nil
}

func (c *Client) GetServerInfo() error {
	params := c.buildParams(map[string]string{})
	requestUrl := fmt.Sprintf("%s/rest/ping?%s", c.BaseURL, params.Encode())
	resp, err := http.Get(requestUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
func (c *Client) SearchSongs(query string) ([]Song, error) {
	params := c.buildParams(map[string]string{
		"query":     query,
		"songCount": "10",
	})

	resp, err := http.Get(fmt.Sprintf("%s/rest/search3.view?%s", c.BaseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		SubsonicResponse struct {
			SearchResult3 struct {
				Songs []Song `json:"song"`
			} `json:"searchResult3"`
		} `json:"subsonic-response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.SubsonicResponse.SearchResult3.Songs, nil
}

func (c *Client) GetPlayURL(songID string) string {
	params := c.buildParams(map[string]string{
		"id":     songID,
		"format": "mp3",
	})
	return fmt.Sprintf("%s/rest/stream.view?%s", c.BaseURL, params.Encode())
}
