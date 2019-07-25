package main

import "testing"


func TestClient(t *testing.T) {
	client := newClient()

	if client.PingTiller() != nil {
		t.Errorf("Excpected a successfull Ping but gut an error")
	}
}