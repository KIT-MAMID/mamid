package main

import (
	"fmt"
	"net/smtp"
)

type Notifier interface {
	SendProblem(problem Problem) error
}

type EmailNotifier struct {
	Contacts []*EmailContact
	Relay    SMTPRelay
}

func (n *EmailNotifier) SendProblem(problem Problem) error {
	content := "A Problem occured: " + problem.Description + "\r\n"
	if problem.ReplicaSet != nil {
		content += fmt.Sprintf("Replica Set id: %d \r\n", *problem.ReplicaSet)
	}
	if problem.Slave != nil {
		content += fmt.Sprintf("Slave id: %d \r\n", *problem.Slave)
	}
	content += "Detailed Description: " + problem.LongDescription + "\r\n"
	subject := "Subject: [MAMID] Problem: " + problem.Description
	msg := []byte("From: " + n.Relay.MailFrom + "\r\n" +
		subject + "\r\n" +
		content)
	return n.sendMailToContacts(msg)
}

func (n *EmailNotifier) sendMailToContacts(msg []byte) error {
	var to []string
	for i := 0; i < len(n.Contacts); i++ {
		to = append(to, n.Contacts[i].Address)
	}
	err := smtp.SendMail(
		n.Relay.Hostname,
		nil,
		n.Relay.MailFrom,
		to,
		msg)
	return err
}
