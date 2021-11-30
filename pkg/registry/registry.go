package registry

import (
	"errors"
	"os"
	"time"
)

// RegistryType is an enum-like type that represents the type of registry
// useful to conditionally observe the registry according to its type
type RegistryType int

const (
	ConstantRegistryType RegistryType = iota
	RemoteRegistryType
)

// Registry is the interface for the JSON registry
// It is used to get the JSON bytes representing a set of endpoints to redirects
type Registry interface {
	RegistryType() RegistryType
	GetJSON() ([]byte, error)
}

type ObserveRegistryResult struct {
	Json []byte
	Err  error
}

// Observe the registry for changes
// returns json bytes and errors via channels
func Observe(registry Registry, period time.Duration) (<-chan ObserveRegistryResult, func()) {
	resultChan := make(chan ObserveRegistryResult)
	
	ticker := time.NewTicker(period)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			
			case <-ticker.C:
				json, err := registry.GetJSON()
				resultChan <- ObserveRegistryResult{json, err}
			}
		}
	}()

	return resultChan, func() {
		ticker.Stop()
		done <- struct{}{}
		close(resultChan)
	}
}

// ConstantRegistry is a registry that always returns the same JSON value
// it can be created from a JSON string or from a file path
type ConstantRegistry struct {
	json []byte
}

func (c *ConstantRegistry) RegistryType() RegistryType {
	return ConstantRegistryType
}

func (c *ConstantRegistry) GetJSON() ([]byte, error) {
	return c.json, nil
}

func newConstantRegistry(json []byte) *ConstantRegistry {
	return &ConstantRegistry{json}
}

func newConstantRegistryFromFilePath(source string) (*ConstantRegistry, error) {
	json, err := fetchFromFilePath(source)
	if err != nil {
		return nil, err
	}
	return newConstantRegistry(json), nil
}

// RemoteRegistry represents a registry that is fetched from a remote URL
// it can be created from a valid URL returning the JSON value on http GET request
type RemoteRegistry struct {
	url string
}

func (r *RemoteRegistry) RegistryType() RegistryType {
	return RemoteRegistryType
}

func (r *RemoteRegistry) GetJSON() ([]byte, error) {
	return fetchFromRemoteURL(r.url)
}

func newRemoteRegistryFromURL(url string) *RemoteRegistry {
	return &RemoteRegistry{url}
}

// getRegistry will check if the given string is a) a JSON by itself b) if is a path to a file c) remote url
func NewRegistry(source string) (Registry, error) {
	// check if it is a json the given source already
	if isArrayOfObjectsJSON(source) {
		return newConstantRegistry([]byte(source)), nil
	}

	// check if is a valid URL
	if isValidURL(source) {
		return newRemoteRegistryFromURL(source), nil
	}

	// in the end check if is a path to a file. If it exists try to read
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		return newConstantRegistryFromFilePath(source)
	}

	return nil, errors.New("source must be either a valid JSON string, a remote URL or a valid path to a JSON file")
}