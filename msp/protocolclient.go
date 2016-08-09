package msp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type MSPClient interface {
	RequestStatus(Target HostPort) ([]Mongod, *Error)
	EstablishMongodState(Target HostPort, m Mongod) *Error
}

type MSPClientImpl struct {
	httpClient http.Client
}

func communicationErrorFromError(err error) *Error {
	return &Error{
		Identifier:      CommunicationError,
		Description:     "Error communicating with slave.",
		LongDescription: err.Error(),
	}
}

func (c MSPClientImpl) RequestStatus(Target HostPort) ([]Mongod, *Error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s:%d/msp/status", Target.Hostname, Target.Port))
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			var result []Mongod
			decodeErr := json.NewDecoder(resp.Body).Decode(&result)
			if decodeErr != nil {
				return nil, communicationErrorFromError(decodeErr)
			}
			// TODO validation
			return result, nil
		} else {
			var slaveError Error
			decodeErr := json.NewDecoder(resp.Body).Decode(&slaveError)
			if decodeErr != nil {
				return nil, communicationErrorFromError(decodeErr)
			} else if validationErr := slaveError.validateFields(); validationErr != nil {
				return nil, communicationErrorFromError(validationErr)
			}
			return nil, &slaveError
		}
	} else {
		return nil, communicationErrorFromError(err)
	}
}

func (c MSPClientImpl) EstablishMongodState(Target HostPort, m Mongod) *Error {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(m)
	if err != nil {
		log.Printf("msp: error serialzing monogd: %s", err)
		panic(err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/msp/establishMongodState", Target.Hostname, Target.Port), buffer)
	if err != nil {
		log.Printf("msp: error creating request object for monogd: %s", err)
		panic(err)
	}

	resp, err := c.httpClient.Do(req)

	if err == nil {
		if resp.StatusCode == http.StatusOK {
			return nil
		} else {
			var slaveError Error
			decodeErr := json.NewDecoder(resp.Body).Decode(&slaveError)
			if decodeErr != nil {
				return communicationErrorFromError(decodeErr)
			}
			return &slaveError
		}
	} else {
		return communicationErrorFromError(err)
	}
}
