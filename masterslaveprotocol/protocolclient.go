package masterslaveprotocol

import (
	"net/http"
	"encoding/json"
	"fmt"
)

type MSPError struct {
	error_message string
}

func (e MSPError) Error() string {
	return e.error_message
}

type MSPClient struct {
	target HostPort
	httpClient http.Client
}

func NewMSPClient(target HostPort) *MSPClient {
	return &MSPClient{target: target}
}

func (c MSPClient) MspSetDataPath(path string) error {
	return MSPError{"Not implemented"}
}

func (c MSPClient) MspStatusRequest() ([]Mongod, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s:%d/msp/status", c.target.Hostname, c.target.Port))
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			var result []Mongod
			json.NewDecoder(resp.Body).Decode(&result)
			return result, nil
		} else {
			return nil, MSPError{fmt.Sprintf("HTTP error %s", resp.Status)}
		}
	} else {
		return nil, MSPError{err.Error()}
	}
}

func (c MSPClient) MspEstablishMongodState(m Mongod) error {
	return MSPError{"Not implemented"}
}