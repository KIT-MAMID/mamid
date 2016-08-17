package main

import (
	"fmt"
	"github.com/vaughan0/go-ini"
	"log"
)

type Parser struct {
}

func (p *Parser) Parse(path string) ([]Contact, error) {
	file, err := ini.LoadFile(path)
	if err != nil {
		return nil, err
	}
	var contacts []Contact
	for name, section := range file {
		for key, value := range section {
			switch key {
			case "email":
				var newContact EmailContact
				newContact.Address = value
				newContact.Name = name
				contacts = append(contacts, newContact)
			default:
				log.Println("Ignoring unknown notifier", key)
			}
		}
	}
	return contacts, err
}

func (p *Parser) ParseConfig(path string) (relay SMTPRelay, apiHost string, contactsFile string, err error) {
	file, err := ini.LoadFile(path)
	if err != nil {
		return
	}
	notifier, ok := file["notifier"]
	if !ok {
		err = fmt.Errorf("Missing 'notifier' section in config file")
		return
	}
	apiHost, ok = notifier["api_host"]
	if !ok {
		err = fmt.Errorf("Missing 'api_host' veriable in 'notifier' section in config file")
		return
	}
	contactsFile, ok = notifier["contacts"]
	if !ok {
		err = fmt.Errorf("Missing 'contacts' veriable in 'notifier' section in config file")
		return
	}

	smtp, ok := file["smtp"]
	if !ok {
		err = fmt.Errorf("Missing 'smtp' section in config file")
		return
	}
	relay.MailFrom, ok = smtp["mail_from"]
	if !ok {
		err = fmt.Errorf("Missing 'mail_from' veriable in 'smtp' section in config file")
		return
	}
	relay.Hostname, ok = smtp["relay_host"]
	if !ok {
		err = fmt.Errorf("Missing 'relay_host' veriable in 'smtp' section in config file")
		return
	}
	return
}
