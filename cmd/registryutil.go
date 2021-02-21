package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func isValidURL(s string) bool {
	_, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	return true
}

func isArrayOfObjectsJSON(s string) bool {
	var js []map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil

}

func fetchFromFilePath(source string) ([]byte, error) {
	data, err := ioutil.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("failed to load file: %w", err)
	}

	if !isArrayOfObjectsJSON(string(data)) {
		return nil, errors.New("invalid JSON from file")
	}

	return data, nil
}

func fetchFromRemoteURL(source string) ([]byte, error) {
	c := http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequest(http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("request URL: %w", err)
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("response URL: %w", err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	if !isArrayOfObjectsJSON(string(body)) {
		return nil, errors.New("invalid JSON from URL")
	}

	return body, nil
}

func registryJSONToRedirects(registryJSON []byte) ([]string, error) {
	var data []map[string]string
	err := json.Unmarshal(registryJSON, &data)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	redirects := make([]string, 0)
	for _, v := range data {
		if strings.Contains(v["endpoint"], "onion") {
			redirects = append(redirects, v["endpoint"])
		}
	}
	if len(redirects) == 0 {
		return nil, errors.New("no valid onion endpoints found")
	}

	return redirects, nil
}
