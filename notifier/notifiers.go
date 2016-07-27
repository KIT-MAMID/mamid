package main

import(
	"net/smtp"
	"fmt"
//	"bytes"
	"log"
	"strings"
)
type Notifier interface {
	SendProblem(problem Problem) error
}

type EmailContact struct {
	Address string
}
type EmailNotifier struct {
	Contacts []*EmailContact
}
func (n *EmailNotifier) SendProblem(problem Problem) error {
	msg := []byte("To: niklas.fuhrberg@gmx.de\r\n"+
                        "From: kit.mamid@gmail.com\r\n"+
                        "Subject: TESST123\r\n")
	fmt.Println("msg")
	n.sendMailToContacts(msg)
	return nil
}
func (n *EmailNotifier) sendMailToContacts(msg []byte) error{
	var auth = smtp.PlainAuth("", "kit.mamid@gmail.com", "uwsngsdlsnh", "smtp.gmail.com")
	var err = smtp.SendMail(
		"smtp.gmail.com:465",
		auth,
		"kit.mamid@gmail.com",
		[]string{"niklas.fuhrberg@gmx.de"},
		msg)
	if err != nil{
		log.Printf("sendSmtp: failure: %q", strings.Split(err.Error(), "\n"))
	}
	return nil
}

