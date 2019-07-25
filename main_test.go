package main

import "testing"


func TestClient(t *testing.T) {
	client, err := newClient()

	if err != nil {
		t.Errorf("Excpected to get *helm.Client but got intead an Error: %s", err.Error())
		return
	}

	if client.PingTiller() != nil {
		t.Errorf("Excpected a successfull Ping but gut an error")
	}
}

