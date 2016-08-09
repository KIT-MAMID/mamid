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
	content := ("A Problem occured: " + problem.Description + "\r\n" +
		"ReplicaSet: " + fmt.Sprint(problem.ReplicaSet) + "\r\n" +
		"Slave: " + fmt.Sprint(problem.Slave) + "\r\n" +
		"long Description:" + problem.LongDescription) + "\r\n"
	subject := ("Subject:" + "KIT-MAMID: Problem in " + fmt.Sprint(problem.ReplicaSet) + "/" + fmt.Sprint(problem.Slave))
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
