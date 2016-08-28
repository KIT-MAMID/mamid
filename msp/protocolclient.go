package msp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"net/http"
)

var mspLog = logrus.WithField("module", "msp")

type MSPClient interface {
	RequestStatus(Target HostPort) ([]Mongod, *Error)
	EstablishMongodState(Target HostPort, m Mongod) *Error
}

type MSPClientImpl struct {
	HttpClient http.Client
}

func communicationErrorFromError(err error) *Error {
	return &Error{
		Identifier:      CommunicationError,
		Description:     "Error communicating with slave.",
		LongDescription: err.Error(),
	}
}

func (c MSPClientImpl) RequestStatus(Target HostPort) ([]Mongod, *Error) {
	resp, err := c.HttpClient.Get(fmt.Sprintf("http://%s:%d/msp/status", Target.Hostname, Target.Port))
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

func (c MSPClientImpl) establishMongodState_validate(targetSlave HostPort, m Mongod) *Error {
	if m.State == MongodStateRecovering {
		return &Error{BadStateDescription, "Invalid Mongod state", fmt.Sprintf("Mongod state `%s` can only by received, not established.", m.State)}
	}
	return nil
}

func (c MSPClientImpl) EstablishMongodState(target HostPort, m Mongod) *Error {

	if err := c.establishMongodState_validate(target, m); err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(m)
	if err != nil {
		mspLog.Errorf("msp: error serialzing monogd: %s", err)
		panic(err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/msp/establishMongodState", target.Hostname, target.Port), buffer)
	if err != nil {
		mspLog.Errorf("msp: error creating request object for monogd: %s", err)
		panic(err)
	}

	resp, err := c.HttpClient.Do(req)

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
