package msp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type MSPClient struct {
	target     HostPort
	httpClient http.Client
}

func (c MSPClient) RequestStatus() ([]Mongod, Error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s:%d/msp/status", c.target.Hostname, c.target.Port))
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			var result []Mongod
			json.NewDecoder(resp.Body).Decode(&result) //TODO Check decode error
			return result, nil
		} else {
			var slaveError SlaveError
			json.NewDecoder(resp.Body).Decode(slaveError) //TODO Check decode error
			return nil, slaveError
		}
	} else {
		return nil, CommunicationError{err.Error()}
	}
}

func (c MSPClient) EstablishMongodState(m Mongod) Error {
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(m)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/msp/establishMongodState", c.target.Hostname, c.target.Port), buffer)
	resp, err := c.httpClient.Do(req)

	if err == nil {
		if resp.StatusCode == http.StatusOK {
			return nil
		} else {
			var slaveError SlaveError
			json.NewDecoder(resp.Body).Decode(slaveError) //TODO Check decode error
			return slaveError
		}
	} else {
		return CommunicationError{err.Error()}
	}
}
