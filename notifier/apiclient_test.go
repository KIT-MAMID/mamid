package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createAPIMock(retCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(retCode)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"id":1,"description":"Slave testneu is unreachable",
		"long_description":"","first_occurred":"2016-08-01T14:24:12.681005208+02:00",
		"last_updated":"2016-08-10T19:33:47.871593537+02:00","slave_id":1,"replica_set_id":0},
		{"id":2,"description":"Slave test4 is unreachable","long_description":"",
		"first_occurred":"2016-08-06T14:46:18.149470116+02:00",
		"last_updated":"2016-08-06T20:39:53.267461615+02:00","slave_id":2,"replica_set_id":0}]`)
	}))
}

func TestApiClientSuccess(t *testing.T) {
	server := createAPIMock(200)
	defer server.Close()
	var client APIClient
	problems, err := client.Receive(server.Listener.Addr().String())
	assert.NoError(t, err)
	assert.Equal(t, len(problems), 2)
}

func TestApiClientFail(t *testing.T) {
	server := createAPIMock(500)
	defer server.Close()
	var client APIClient
	problems, err := client.Receive(server.Listener.Addr().String())
	assert.Error(t, err)
	assert.Equal(t, problems, []Problem(nil))
}
