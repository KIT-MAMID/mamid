package msp

import (
	"net/http"
	"encoding/json"
	"bytes"
	"fmt"
)

type CommunicationError struct {
	error_message string
}

func (e CommunicationError) Error() string {
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
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(path)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/msp/setDataPath", c.target.Hostname, c.target.Port), buffer)
	resp, err := c.httpClient.Do(req)

	if err == nil {
		if resp.StatusCode == http.StatusOK {
			return nil
		} else {
			return MSPErrorFromJson(resp.Body)
		}
	} else {
		return CommunicationError{err.Error()}
	}
}

func (c MSPClient) MspStatusRequest() ([]Mongod, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s:%d/msp/status", c.target.Hostname, c.target.Port))
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			var result []Mongod
			json.NewDecoder(resp.Body).Decode(&result) //TODO Check decode error
			return result, nil
		} else {
			var mspError MSPError
			json.NewDecoder(resp.Body).Decode(mspError) //TODO Check decode error
			return nil, mspError
		}
	} else {
		return nil, CommunicationError{err.Error()}
	}
}

func (c MSPClient) MspEstablishMongodState(m Mongod) error {
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(m)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/msp/establishMongodState", c.target.Hostname, c.target.Port), buffer)
	resp, err := c.httpClient.Do(req)

	if err == nil {
		if resp.StatusCode == http.StatusOK {
			return nil
		} else {
			var mspError MSPError
			json.NewDecoder(resp.Body).Decode(mspError) //TODO Check decode error
			return mspError
		}
	} else {
		return CommunicationError{err.Error()}
	}
}