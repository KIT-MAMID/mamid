package main

type Notifier interface {
	SendProblem(problem Problem) error
}

type EmailNotifier struct {
	Contacts []*EmailContact
}

func (n *EmailNotifier) SendProblem(problem *Problem) error {
	return nil
}

func (n *EmailNotifier) sendMailToContacts(msg string) error {
	return nil
}
