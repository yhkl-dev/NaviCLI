package subsonic

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func (c *Client) GetRandomSongs(size int) ([]Song, error) {
	if size <= 0 {
		size = c.PageSize
	}
	params, err := c.buildParams(map[string]string{
		"size":   fmt.Sprintf("%d", size),
		"format": "json",
	})
	if err != nil {
		return nil, fmt.Errorf("build params: %w", err)
	}
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
	params, err := c.buildParams(map[string]string{})
	if err != nil {
		return fmt.Errorf("build params: %w", err)
	}
	requestUrl := fmt.Sprintf("%s/rest/ping.view?%s", c.BaseURL, params.Encode())

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, response: %s", resp.StatusCode, string(body))
	}

	// Consume body for connection reuse
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *Client) SearchSongs(query string) ([]Song, error) {
	params, err := c.buildParams(map[string]string{
		"query":     query,
		"songCount": "10",
	})
	if err != nil {
		return nil, fmt.Errorf("build params: %w", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/search3.view?%s", c.BaseURL, params.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, response: %s", resp.StatusCode, string(body))
	}

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
	params, err := c.buildParams(map[string]string{
		"id":     songID,
		"format": "mp3",
	})
	if err != nil {
		log.Printf("GetPlayURL buildParams error: %v", err)
		return ""
	}
	return fmt.Sprintf("%s/rest/stream.view?%s", c.BaseURL, params.Encode())
}

func (c *Client) GetAlbumList2(albumType string, size int) ([]AlbumID3, error) {
	if size <= 0 {
		size = 20
	}
	params, err := c.buildParams(map[string]string{
		"type": albumType,
		"size": fmt.Sprintf("%d", size),
	})
	if err != nil {
		return nil, fmt.Errorf("build params: %w", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/getAlbumList2?%s", c.BaseURL, params.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		SubsonicResponse struct {
			AlbumList2 struct {
				Albums []AlbumID3 `json:"album"`
			} `json:"albumList2"`
		} `json:"subsonic-response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.SubsonicResponse.AlbumList2.Albums, nil
}

func (c *Client) GetAlbum(albumID string) ([]Song, error) {
	params, err := c.buildParams(map[string]string{
		"id": albumID,
	})
	if err != nil {
		return nil, fmt.Errorf("build params: %w", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/getAlbum?%s", c.BaseURL, params.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		SubsonicResponse struct {
			Album struct {
				Songs []Song `json:"song"`
			} `json:"album"`
		} `json:"subsonic-response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.SubsonicResponse.Album.Songs, nil
}

func (c *Client) GetCoverArtURL(coverArtID string) string {
	if coverArtID == "" {
		return ""
	}
	params, err := c.buildParams(map[string]string{
		"id":   coverArtID,
		"size": "300",
	})
	if err != nil {
		log.Printf("GetCoverArtURL buildParams error: %v", err)
		return ""
	}
	return fmt.Sprintf("%s/rest/getCoverArt.view?%s", c.BaseURL, params.Encode())
}
