package main

import (
	"encoding/base64"
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
	subject := "[MAMID] Problem: " + problem.Description
	subject = "Subject: =?utf-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
	msg := "From: " + n.Relay.MailFrom + "\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-transfer-encoding: binary\r\n" +
		subject + "\r\n\r\n" +
		content
	return n.sendMailToContacts(msg)
}

func (n *EmailNotifier) sendMailToContacts(msg string) error {
	for i := 0; i < len(n.Contacts); i++ {
		err := smtp.SendMail(
			n.Relay.Hostname,
			nil,
			n.Relay.MailFrom,
			[]string{n.Contacts[i].Address},
			[]byte("To: "+n.Contacts[i].Address+"\r\n"+msg))
		if err != nil {
			return err
		}
	}

	return nil
}
