package subsonic

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Init(baseUrl, username, password, clientId, apiVersion string) *Client {
	httpClient := &http.Client{}
	client := &Client{
		BaseURL:    baseUrl,
		Username:   username,
		Password:   password,
		ClientID:   clientId,
		APIVersion: apiVersion,
		HttpClient: httpClient,
	}
	return client
}

func (c *Client) authToken(password string) (string, string) {
	salt := randSeq(8)
	token := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))

	return token, salt
}

func (c *Client) buildParams(extraParams map[string]string) url.Values {
	salt := "randomsalt"
	token, salt := c.authToken(c.Password)
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
