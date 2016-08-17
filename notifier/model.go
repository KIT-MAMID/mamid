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
	Id              uint
	Description     string
	LongDescription string
	FirstOccured    time.Time
	LastUpdate      time.Time
	Slave           *uint
	ReplicaSet      *uint
}
