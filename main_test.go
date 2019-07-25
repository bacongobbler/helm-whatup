package main

import (
	"bytes"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func TestClient(t *testing.T) {
	client, err := newClient()

	if err != nil {
		t.Errorf("Excpected to get *helm.Client but got intead an Error: %s", err.Error())
	}

	if client.PingTiller() != nil {
		t.Errorf("Excpected a successfull Ping but gut an error")
	}
}

func TestCharts(t *testing.T) {
	// 10 released/ installed Charts
//	var expectedResults = []byte(`[
//	{
//		"releaseName": "coredns",
//		"chartName": "coredns",
//		"installedVersion": "1.5.0",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "hunter",
//		"chartName": "karma",
//		"installedVersion": "1.1.5",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "jenkins",
//		"chartName": "jenkins",
//		"installedVersion": "0.32.1",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "kafka-manager",
//		"chartName": "kafka-manager",
//		"installedVersion": "1.1.1",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "kapacitor",
//		"chartName": "kapacitor",
//		"installedVersion": "0.3.0",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "kube-hunter",
//		"chartName": "kube-hunter",
//		"installedVersion": "1.0.0",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "kube-slack",
//		"chartName": "kube-slack",
//		"installedVersion": "0.1.0",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "kuberhealthy",
//		"chartName": "kuberhealthy",
//		"installedVersion": "1.1.1",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "lamp",
//		"chartName": "lamp",
//		"installedVersion": "0.1.2",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "luigi",
//		"chartName": "luigi",
//		"installedVersion": "2.7.2",
//		"status": "OUTDATED"
//	},
//	{
//		"releaseName": "magento",
//		"chartName": "magento",
//		"installedVersion": "0.4.10",
//		"status": "OUTDATED"
//	}
//]`)

	client, err := newClient()

	if err != nil {
		t.Errorf("Excpected to get *helm.Client but got intead an Error: %s", err.Error())
	}

	releases, err := fetchReleases(client)
	if err != nil {
		t.Errorf("Excpected to get []*release.Release but got an Error instead: %s", err.Error())
	}

	repositories, err := fetchIndices(client)
	if err != nil {
		t.Errorf("Excpected to get []*repo.IndexFile but got an Error instead: %s", err.Error())
	}

	result, err := parseReleases(releases, repositories)
	if err != nil {
		t.Errorf("Excpected to get []ChartVersionInfo but got an Error instead: %s", err.Error())
	}

	outputFormat = "json"
	output := []byte(captureOutput(func() {
		err = formatOutput(result)
		if err != nil {
			t.Errorf("Excpected to get nothing but got an Error instead: %s", err.Error())
		}
	}))

	// remove [].{}.latestVersion
	newJSON := output
	manipulatedOutput := gjson.GetBytes(output, "#.latestVersion")

	for i := range manipulatedOutput.Array() {
		newJSON, err = sjson.DeleteBytes(newJSON, strconv.Itoa(i)+".latestVersion")
		if err != nil {
			t.Errorf("[FATAL] Internal Error while manipulating JSON String: %s", err.Error())
		}
	}

	// compare JSON Strings
	// TODO: Compare Objects
	//outputEqual, err := equalJSON(string(expectedResults), string(newJSON))
	//if err != nil {
	//	t.Errorf("[FATAL] Internal Error while comparing Output and expectation: %s", err.Error())
	//}
//
//	if !outputEqual {
//		t.Errorf("Expected Output is equal with expectation but got 'false'!")
//	}
}

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	f()

	log.SetOutput(os.Stderr)
	return buf.String()
}