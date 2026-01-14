package subsonic

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randSeq(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	result := make([]byte, n)
	for i, v := range b {
		result[i] = letters[int(v)%len(letters)]
	}
	return string(result), nil
}

func Init(baseUrl, username, password, clientId, apiVersion string, pageSize int, httpTimeout time.Duration) *Client {
	client := &Client{
		BaseURL:    baseUrl,
		Username:   username,
		Password:   password,
		ClientID:   clientId,
		APIVersion: apiVersion,
		PageSize:   pageSize,
		HttpClient: &http.Client{Timeout: httpTimeout},
	}
	return client
}

func (c *Client) authToken(password string) (string, string, error) {
	salt, err := randSeq(8)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate salt: %w", err)
	}
	token := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))

	return token, salt, nil
}

func (c *Client) buildParams(extraParams map[string]string) url.Values {
	token, salt, err := c.authToken(c.Password)
	if err != nil {
		// In Phase 3 refactoring, this will return error properly
		// For now, panic to maintain current function signature
		panic(fmt.Sprintf("authentication failed: %v", err))
	}
	params := url.Values{}
	params.Add("u", c.Username)
	params.Add("t", token)
	params.Add("s", salt)
	params.Add("v", c.APIVersion)
	params.Add("c", c.ClientID)
	params.Add("f", "json")

	for k, v := range extraParams {
		params.Add(k, v)
	}
	return params
}
