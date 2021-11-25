package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

func isArrayOfObjectsJSON(s string) bool {
	var js []map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil

}

func isValidURL(s string) bool {
	_, err := url.ParseRequestURI(s)
	return err == nil
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

