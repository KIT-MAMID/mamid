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

func (apiclient *APIClient) Receive(host string) []Problem {
	var problems []Problem
	resp, err := apiclient.httpClient.Get(fmt.Sprintf("http://%s/api/problems", host))
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			json.NewDecoder(resp.Body).Decode(&problems) //TODO Check decode error
		} else {
			//TODO handle error
		}
	} else {
		return nil
	}
	return problems
}
