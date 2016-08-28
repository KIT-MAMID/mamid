package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type APIClient struct {
	httpClient http.Client
}

func (apiclient *APIClient) Receive(host string) (problems []Problem, err error) {
	resp, err := apiclient.httpClient.Get(fmt.Sprintf("%s/api/problems", host))
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp.Body).Decode(&problems)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("API returned non 200 %d", resp.StatusCode)
		}
	}
	return
}
