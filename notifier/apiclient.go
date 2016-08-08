package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type APIClient struct {
	httpClient http.Client
}

type Problem struct {
	Id              uint
	Description     string
	LongDescription string
	FirstOccured    time.Time
	LastUpdate      time.Time
	Slave           uint
	ReplicaSet      uint
}

func (apiclient *APIClient) Receive(host string) (problems []Problem, err error) {
	resp, err := apiclient.httpClient.Get(fmt.Sprintf("http://%s/api/problems", host))
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp.Body).Decode(&problems)
			if err != nil {
				return nil, err;
			}
		} else {
			return nil, fmt.Errorf("API returned non 200 %d", resp.StatusCode)
		}
	} else {
		return nil, err
	}
	return problems, nil
}
