package main

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestConfigFile(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "mamid_test")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[smtp]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("relay_host=localhost:25\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("mail_form=test@localhost\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[notifier]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("api_host=localhost:8080\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("contacts=contacts.ini\n")
	assert.NoError(t, err)
	tmpFile.Sync()
	tmpFile.Close()

	var p Parser
	relay, apiHost, contactsFile, err := p.ParseConfig(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, contactsFile, "contacts.ini")
	assert.Equal(t, apiHost, "localhost:8080")
	assert.Equal(t, relay.Hostname, "localhost:25")
	assert.Equal(t, relay.MailFrom, "test@localhost")
}

func TestConfigFileMissingFile(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "mamid_test")
	assert.NoError(t, err)
	os.Remove(tmpFile.Name())

	var p Parser
	_, _, _, err = p.ParseConfig(tmpFile.Name())
	assert.Error(t, err)
}

func TestConfigFileMissingConf(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "mamid_test")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[smtp]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("relay_host=localhost:25\n")
	assert.NoError(t, err)
	//_, err = tmpFile.WriteString("mail_form=test@localhost\n")
	//assert.NoError(t, err)
	_, err = tmpFile.WriteString("[notifier]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("api_host=localhost:8080\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("contacts=contacts.ini\n")
	assert.NoError(t, err)
	tmpFile.Sync()
	tmpFile.Close()

	var p Parser
	_, _, _, err = p.ParseConfig(tmpFile.Name())
	assert.Error(t, err)
}

func TestConfigFileMissingSection(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "mamid_test")
	assert.NoError(t, err)
	//_, err = tmpFile.WriteString("[smtp]\n")
	//assert.NoError(t, err)
	_, err = tmpFile.WriteString("relay_host=localhost:25\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("mail_form=test@localhost\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[notifier]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("api_host=localhost:8080\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("contacts=contacts.ini\n")
	assert.NoError(t, err)
	tmpFile.Sync()
	tmpFile.Close()

	var p Parser
	_, _, _, err = p.ParseConfig(tmpFile.Name())
	assert.Error(t, err)
}

func TestContactsFile(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "mamid_test")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[hans]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("email=hans@localhost\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[peter]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("email=peter@localhost\n\n")
	assert.NoError(t, err)
	tmpFile.Sync()
	tmpFile.Close()

	var p Parser
	contacts, err := p.Parse(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, len(contacts), 2)
	for i := 0; i < len(contacts); i++ {
		assert.IsType(t, EmailContact{}, contacts[i])
		assert.NotEmpty(t, contacts[i])
	}
}

func TestContactsFileAdditionalType(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "mamid_test")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[hans]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("email=hans@localhost\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("[peter]\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("email=peter@localhost\n\n")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("jabber=peter@localhost\n\n")
	assert.NoError(t, err)
	tmpFile.Sync()
	tmpFile.Close()

	var p Parser
	contacts, err := p.Parse(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, len(contacts), 2)
	for i := 0; i < len(contacts); i++ {
		assert.IsType(t, EmailContact{}, contacts[i])
		assert.NotEmpty(t, contacts[i])
	}
}

func TestContactsFileNoFile(t *testing.T) {
	var p Parser
	_, err := p.Parse("")
	assert.Error(t, err)
}
