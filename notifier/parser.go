package main

import (
	"fmt"
	"github.com/vaughan0/go-ini"
)

type Config struct {
	relay                                            SMTPRelay
	apiHost, contactsFile, masterCA, apiCert, apiKey string
}

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
				log.Infof("Ignoring unknown notifier `%s`", key)
			}
		}
	}
	return contacts, err
}

func (p *Parser) ParseConfig(path string) (config Config, err error) {
	file, err := ini.LoadFile(path)
	if err != nil {
		return
	}
	// Notifier section
	notifier, ok := file["notifier"]
	if !ok {
		err = fmt.Errorf("Missing 'notifier' section in config file")
		return
	}
	config.apiHost, ok = notifier["api_host"]
	if !ok {
		err = fmt.Errorf("Missing 'api_host' veriable in 'notifier' section in config file")
		return
	}
	config.apiKey, _ = notifier["api_key"]
	config.apiCert, _ = notifier["api_cert"]
	if tmp := config.apiCert + config.apiKey; tmp != "" && (config.apiCert == "" || config.apiKey == "") {
		err = fmt.Errorf("Both, `api_key` and `api_cert` have to be defined in the 'notifier' section in config file")
		return
	}
	config.masterCA, ok = notifier["master_ca"]

	config.contactsFile, ok = notifier["contacts"]
	if !ok {
		err = fmt.Errorf("Missing 'contacts' veriable in 'notifier' section in config file")
		return
	}

	// SMTP section
	smtp, ok := file["smtp"]
	if !ok {
		err = fmt.Errorf("Missing 'smtp' section in config file")
		return
	}
	config.relay.MailFrom, ok = smtp["mail_from"]
	if !ok {
		err = fmt.Errorf("Missing 'mail_from' veriable in 'smtp' section in config file")
		return
	}
	config.relay.Hostname, ok = smtp["relay_host"]
	if !ok {
		err = fmt.Errorf("Missing 'relay_host' veriable in 'smtp' section in config file")
		return
	}
	return
}
