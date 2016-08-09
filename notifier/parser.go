package main

import (
	"github.com/vaughan0/go-ini"
)

type Parser struct {
}

func (p *Parser) Parse(path string) ([]Contact, error) {
	file, err := ini.LoadFile(path)
	var contacts []Contact
	for name, section := range file {
		for key, value := range section {
			switch key {
			case "email":
				var newContact EmailContact
				newContact.Address = value
				newContact.Name = name
				contacts = append(contacts, newContact)
				email.Contacts = append(email.Contacts, &newContact)
			default:
				panic("unrecknoized input")
				return nil, err
			}
		}
	}
	return contacts, nil
}
