package main

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
