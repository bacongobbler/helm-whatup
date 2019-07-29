// Copyright (c) 2019 FABMation GmbH
// All Rights Reserved
package main

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"reflect"
	"strconv"
	"testing"
)

func TestClient(t *testing.T) {
	client, err := newClient()

	if err != nil {
		t.Errorf("Excpected to get *helm.Client but got intead an Error: %s", err.Error())
		return
	}

	if err = client.PingTiller(); err != nil {
		t.Errorf("Excpected a successfull Ping but gut an error: %s", err.Error())
		return
	}
}

func TestCharts(t *testing.T) {
	// 10 released/ installed Charts
	var expectedResults = []byte(`[
	{
		"releaseName": "coredns",
		"chartName": "coredns",
		"installedVersion": "1.5.0",
		"status": "OUTDATED"
	},
	{
		"releaseName": "hunter",
		"chartName": "karma",
		"installedVersion": "1.1.5",
		"status": "OUTDATED"
	},
	{
		"releaseName": "jenkins",
		"chartName": "jenkins",
		"installedVersion": "0.32.1",
		"status": "OUTDATED"
	},
	{
		"releaseName": "kafka-manager",
		"chartName": "kafka-manager",
		"installedVersion": "1.1.1",
		"status": "OUTDATED"
	},
	{
		"releaseName": "kapacitor",
		"chartName": "kapacitor",
		"installedVersion": "0.3.0",
		"status": "OUTDATED"
	},
	{
		"releaseName": "kube-hunter",
		"chartName": "kube-hunter",
		"installedVersion": "1.0.0",
		"status": "OUTDATED"
	},
	{
		"releaseName": "kube-slack",
		"chartName": "kube-slack",
		"installedVersion": "0.1.0",
		"status": "OUTDATED"
	},
	{
		"releaseName": "kuberhealthy",
		"chartName": "kuberhealthy",
		"installedVersion": "1.1.1",
		"status": "OUTDATED"
	},
	{
		"releaseName": "lamp",
		"chartName": "lamp",
		"installedVersion": "0.1.2",
		"status": "OUTDATED"
	},
	{
		"releaseName": "luigi",
		"chartName": "luigi",
		"installedVersion": "2.7.2",
		"status": "OUTDATED"
	},
	{
		"releaseName": "magento",
		"chartName": "magento",
		"installedVersion": "0.4.10",
		"status": "OUTDATED"
	}
	]`)

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
	formatOutputReturn = true
	logDebug = true // Enable Debug Output
	output, err := formatOutput(result)
	if err != nil {
		t.Errorf("Excpected to get JSON Output but got an Error instead: %s", err.Error())
	}

	outputFormats := []string{
		"table",
		"plain",
		"yml",
		"yaml",
	}
	formatOutputReturn = false
	for _, format := range outputFormats {
		outputFormat = format
		_, err = formatOutput(result)

		if err != nil {
			t.Errorf("Excpected to get %s Output but got an Error instead: %s", format, err.Error())
		}
	}

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
	outputEqual, err := equalJSON(string(expectedResults), string(newJSON))
	if err != nil {
		t.Errorf("[FATAL] Internal Error while comparing Output and expectation: %s", err.Error())
	}

	if !outputEqual {
		t.Errorf("Expected Output is equal with expectation but got 'false'!")
	}
}

func TestMain(t *testing.T) {
	main()
}


/// >>>>> Needed Functions <<<<<


// Source: https://gist.github.com/turtlemonvh/e4f7404e28387fadb8ad275a99596f67#file-equal_json-go-L11
func equalJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("error mashalling string 1 :: %s", err.Error())
	}

	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		debug("s2 String: %s", s2)
		return false, fmt.Errorf("error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}