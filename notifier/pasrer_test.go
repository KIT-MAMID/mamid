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
