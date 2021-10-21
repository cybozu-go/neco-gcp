package slacknotifier

import (
	"os"
	"testing"
)

func TestComputeLogInvalid(t *testing.T) {
	for _, n := range []string{
		"./log/invalid/from_cloud_function.json",
		"./log/invalid/operation_first.json",
		"./log/invalid/unsupported_method_name.json",
	} {
		json, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}

		_, err = NewComputeLog(json)
		if err == nil {
			t.Errorf("filename: %s log: %#v", n, err)
		}
	}
}

func TestComputeLogValid(t *testing.T) {
	testCases := []struct {
		inputFileName string
		name          string
		message       string
		method        string
		projectID     string
		zone          string
		timestamp     string
	}{
		{
			"./log/valid/startup_script.json",
			"sample-0",
			"INFO startup-script: startup",
			startupScriptMethodName,
			"neco-dev0",
			"asia-northeast1-a",
			"2020-11-13T08:17:15Z",
		},
		{
			"./log/valid/delete.json",
			"sample-1",
			"Instance Deleted",
			computeDeleteMethodName,
			"neco-dev1",
			"asia-northeast1-b",
			"2020-11-13T06:58:06Z",
		},
		{
			"./log/valid/insert.json",
			"sample-2",
			"Instance Inserted",
			computeInsertMethodName,
			"neco-dev2",
			"asia-northeast1-c",
			"2020-11-13T06:56:27Z",
		},
	}

	for _, tt := range testCases {
		json, err := os.ReadFile(tt.inputFileName)
		if err != nil {
			t.Fatalf("%s: %v", tt.inputFileName, err)
		}

		m, err := NewComputeLog(json)
		if err != nil {
			t.Fatalf("%s: %v", tt.inputFileName, err)
		}

		name := m.GetInstanceName()
		if name != tt.name {
			t.Errorf("expect: %s, actual: %s", tt.name, name)
		}

		method := m.GetMethodName()
		if method != tt.method {
			t.Errorf("expect: %s, actual: %s", tt.method, method)
		}

		msg := m.GetMessage()
		if msg != tt.message {
			t.Errorf("expect: %s, actual: %s", tt.message, msg)
		}

		id := m.GetProjectID()
		if id != tt.projectID {
			t.Errorf("expect: %s, actual: %s", tt.projectID, id)
		}

		zone := m.GetZone()
		if zone != tt.zone {
			t.Errorf("expect: %s, actual: %s", tt.zone, zone)
		}

		ts := m.GetTimeStamp()
		if ts != tt.timestamp {
			t.Errorf("expect: %s, actual: %s", tt.timestamp, ts)
		}
	}
}
