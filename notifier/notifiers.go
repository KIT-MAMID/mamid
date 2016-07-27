package main

import(
	"net/smtp"
	"fmt"
)
type Notifier interface {
	SendProblem(problem Problem) error
}

type EmailNotifier struct {
	Contacts []*EmailContact
}

func (n *EmailNotifier) SendProblem(problem Problem) error {
	content := ("A Problem occured: " + problem.Description + "\r\n" +
		   "ReplicaSet: " + fmt.Sprint(problem.ReplicaSet) + "\r\n" +
		   "Slave: " + fmt.Sprint( problem.Slave) + "\r\n" +
		   "long Description:" + problem.LongDescription) + "\r\n"
	subject := ("Subject:" + "KIT-MAMID: Problem in " + fmt.Sprint(problem.ReplicaSet) + "/" + fmt.Sprint(problem.Slave))
	msg := []byte("To: niklas.fuhrberg@gmx.de\r\n"+
                        "From: kit.mamid@gmail.com\r\n"+
                        subject + "\r\n" +
			content)
	return n.sendMailToContacts(msg)
}
func (n *EmailNotifier) sendMailToContacts(msg []byte) error{
	auth := smtp.PlainAuth("", "kit.mamid@gmail.com", "uwsngsdlsnh", "smtp.gmail.com")
	var to []string
	for i := 0; i < len(n.Contacts); i++{
		to[i] = n.Contacts[i].Address;
	}
	err := smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		"kit.mamid@gmail.com",
		[]string{"niklas.fuhrberg@gmx.de"},
		msg)
	return err
}

