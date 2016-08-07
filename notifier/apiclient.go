package main

import "time"

type APIClient struct {
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
	//[GET]
	return problems
}
