package main

import (
	"time"
)

type Contact interface {
}

type EmailContact struct {
	Name    string
	Address string
}

type SMTPRelay struct {
	Hostname string
	MailFrom string
}

type Problem struct {
	Id              uint      `json:"id"`
	Description     string    `json:"description"`
	LongDescription string    `json:"long_description"`
	FirstOccured    time.Time `json:"first_occured"`
	LastUpdate      time.Time `json:"last_update"`
	Slave           *uint     `json:"slave_id"`
	ReplicaSet      *uint     `json:"replica_set_id"`
}
