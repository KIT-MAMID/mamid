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
	InitiateReplicaSet(Target HostPort, msg RsInitiateMessage) *Error
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
	resp, err := c.HttpClient.Get(fmt.Sprintf("%smsp/status", constructBaseUrl(Target)))
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

	req, err := http.NewRequest("POST", fmt.Sprintf("%smsp/establishMongodState", constructBaseUrl(target)), buffer)
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

func constructBaseUrl(target HostPort) (base string) {
	return fmt.Sprintf("https://%s:%d/", target.Hostname, target.Port)
}

func (c MSPClientImpl) InitiateReplicaSet(target HostPort, msg RsInitiateMessage) *Error {

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(msg)
	if err != nil {
		mspLog.Errorf("msp: error serialzing RsInitiateMessage: %s", err)
		panic(err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%smsp/rsInitiate", constructBaseUrl(target)), buffer)
	if err != nil {
		mspLog.Errorf("msp: error creating request object for rsInitiate: %s", err)
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
