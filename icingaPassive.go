// Copyright 2015 ThoughtWorks, Inc.

// This file is part of getgauge/xml-report.

// getgauge/xml-report is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// getgauge/xml-report is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with getgauge/xml-report.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"time"
	"net"
	"net/http"
	"crypto/tls"
	"bytes"
	"bufio"
	"encoding/json"
	"github.com/getgauge/common"
	"github.com/guandalf/icinga-passive/gauge_messages"
	"github.com/guandalf/icinga-passive/listener"
)

type IcingaPassiveMessage struct {
	PluginOutput     string     `json:"plugin_output"`
	ExitStatus       int        `json:"exit_status"`
	PerformanceData  [1]string  `json:"performance_data"`
}

//Golang has static types and does not like dynamic json
//https://github.com/xert/go-icinga2/blob/master/icinga/hosts.go
//we need to decode the following event messages which may differ based on their selected types
//type: CheckResult
//{"check_result":{"active":true,"check_source":"icinga2","command":["/usr/lib64/nagios/plugins/check_ssh","54.218.59.62"],"execution_end":1476468766.2535429001,"execution_start":1476468756.2439880371,"exit_status":2.0,"output":"CRITICAL - Socket timeout after 10 seconds","performance_data":[],"schedule_end":1476468766.2536571026,"schedule_start":1476468756.2430608273,"state":2.0,"type":"CheckResult","vars_after":{"attempt":1.0,"reachable":false,"state":2.0,"state_type":1.0},"vars_before":{"attempt":1.0,"reachable":false,"state":2.0,"state_type":1.0}},"host":"i-a78837bf","service":"ssh","timestamp":1476468766.2537300587,"type":"CheckResult"}

type EventTypeCheckResult struct {
	CheckResult	CheckResultAttrs `json:"check_result"`
	Host		string		`json:"host,omitempty"`
	Service		string		`json:"service,omitempty"`
	Timestamp	float64		`json:"timestamp,omitempty"`
	Type		string		`json:"type,omitempty"`
}

type PerformanceDataValue struct {
	Counter		bool		`json:"counter,omitempty"`
	Crit		json.Number	`json:"crit,omitempty"`
	Label		string		`json:"label,omitempty"`
	Max		json.Number	`json:"max,omitempty"`
	Min		json.Number	`json:"min,omitempty"`
	Type		string		`json:"type,omitempty"`
	Unit		string		`json:"unit,omitempty"`
	Value		json.Number	`json:"value,omitempty"`
	Warn		json.Number	`json:"warn,omitempty"`
}

type CheckResultAttrs struct {
	Active		bool		`json:"active,omitempty"`
	CheckSource	string		`json:"check_source,omitempty"`
	Command		[]string	`json:"command,omitempty"`
	ExecutionEnd	float64		`json:"execution_end,omitempty"`
	ExecutionStart	float64		`json:"execution_start,omitempty"`
	ExitStatus	json.Number	`json:"exit_status,omitempty"`
	Output		string		`json:"output,omitempty"`
	PerformanceDataParsed []PerformanceDataValue `json:"performance_data,omitempty"`
	PerformanceDataStr []string	`json:"performance_data,omitempty"`
	ScheduleEnd	float64		`json:"schedule_end,omitempty"`
	ScheduleStart	float64		`json:"schedule_start,omitempty"`
	State		json.Number	`json:"state,omitempty"`
	Type		string		`json:"type,omitempty"`
	VarsAfter	*struct {
		Attempt		json.Number	`json:"attempt,omitempty"`
		Reachable	bool		`json:"reachable,omitempty"`
		State		json.Number	`json:"state,omitempty"`
		StateType	json.Number	`json:"state_type,omitempty"`
	} `json:"vars_after,omitempty"`
	VarsBefore	*struct {
		Attempt		json.Number	`json:"attempt,omitempty"`
		Reachable	bool		`json:"reachable,omitempty"`
		State		json.Number	`json:"state,omitempty"`
		StateType	json.Number	`json:"state_type,omitempty"`
	} `json:"vars_before,omitempty"`
}

const (
	EXECUTION_ACTION            = "execution"
	GAUGE_HOST                  = "127.0.0.1"
	GAUGE_PORT_ENV              = "plugin_connection_port"
	PLUGIN_ACTION_ENV           = "icinga-passive_action"
	icingaPassive               = "icinga-passive"
	timeFormat                  = "2006-01-02 15.04.05"
)

var projectRoot string
var pluginDir string
var StartTime int64
var CheckResultCount int64
var CheckResultCountObject map[string]int64

var StateChangesObject map[string][]int64

func createReport(suiteResult *gauge_messages.SuiteExecutionResult) {

	fmt.Println()
	urlBase := os.Getenv("icinga_hostname")
	hostToUpdate := os.Getenv("icinga_host_to_update")
	checkToUpdate := os.Getenv("icinga_check_to_update")
	apiUser := os.Getenv("icinga_username")
	apiPass := os.Getenv("icinga_password")

	var mpd [1]string
	mpd[0] = "result=0;1;2;0;3"
	message := fmt.Sprintf("Error message to be implemented")
	failed :=suiteResult.GetSuiteResult().GetFailed()
	icingaStatusCode := 0
	if failed {
		icingaStatusCode = 2
	}
	m := IcingaPassiveMessage{message, icingaStatusCode , mpd}
	b, err := json.Marshal(m)
	if err == nil {
		fmt.Printf("%s\n", b)
	} else {
		fmt.Println("Error marshaling json")
	}

	eventsEndpoint := urlBase + "/v1/actions/process-check-result?service=" + hostToUpdate + "!" + checkToUpdate
	httpClient := initHTTPClient()

	// var requestBody = []byte(`{ "plugin_output": "<p>This is a string</p>", "exit_status": 0, "performance_data": ["result=0;1;2;0;3"]}`)
	//TODO: Depending on the types the user chooses we need to select our response handling
	fmt.Println("Updating:", eventsEndpoint)
	req, err := http.NewRequest("POST", eventsEndpoint, bytes.NewBuffer(b))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(apiUser, apiPass)

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("Server error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	//http://stackoverflow.com/questions/22108519/how-do-i-read-a-streaming-response-body-using-golangs-net-http-package
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')

		if err != nil {
			fmt.Println("Error reading stream", err)
			return
		}
		//fmt.Println("Processing message")
		//fmt.Println(string(line))

		//call handler for json decoding and message processing
		handleEventTypes(string(line))
	}
}

func createIcingaNotifier() {
	os.Chdir(projectRoot)
	listener, err := listener.NewGaugeListener(GAUGE_HOST, os.Getenv(GAUGE_PORT_ENV))
	if err != nil {
		fmt.Println("Could not create the gauge listener")
		os.Exit(1)
	}
	listener.OnSuiteResult(createReport)
	listener.Start()
}

func findPluginAndProjectRoot() {
	projectRoot = os.Getenv(common.GaugeProjectRootEnv)
	if projectRoot == "" {
		fmt.Printf("Environment variable '%s' is not set. \n", common.GaugeProjectRootEnv)
		os.Exit(1)
	}
	var err error
	pluginDir, err = os.Getwd()
	if err != nil {
		fmt.Printf("Error finding current working directory: %s \n", err)
		os.Exit(1)
	}
}

func displayStates(values []int64, is_host bool) string {
	var str = ""
	for i:= 0; i < len(values); i++ {
		//ugly, but I'm lazy
		if is_host == true {
			if values[i] < 2 {
				str += "\x1b[32;1mUP\x1b[0m"
			} else if values[i] >= 2 {
				str += "\x1b[31;1mDown\x1b[0m"
			}
		} else {
			if values[i] == 0 {
				str += "\x1b[32;1mOK\x1b[0m"
			} else if values[i] == 1 {
				str += "\x1b[33;1mWarning\x1b[0m"
			} else if values[i] == 2 {
				str += "\x1b[31;1mCritical\x1b[0m"
			} else if values[i] == 3 {
				str += "\x1b[35;1mUnknown\x1b[0m"
			}
		}

		str += " "
	}

	return str
}

func detectFlapping(values []int64) bool {
	var oldState int64 = 0
	var newState int64 = 0
	var flappingCount float64 = 0.0

	//require a minimum
	if len(values) < 5 {
		return false
	}

	for i:= 0; i < len(values); i++ {
		if i == 0 {
			newState = values[i]
		} else {
			oldState = values[i-1]
			newState = values[i]
		}

		if oldState != newState {
			flappingCount++
		}
	}

	if flappingCount > float64(len(values) / 3) {
		return true
	}

	return false
}

func initHTTPClient() *http.Client {
	sslSkipVerify := true //TODO: config option

	//https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
	httpTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: time.Second * 5,
			KeepAlive: time.Second * 30,
		}).Dial,
		TLSHandshakeTimeout: time.Second * 5,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: sslSkipVerify,
		},
	}
	//We don't need a client timeout for http long polling!
	httpClient := &http.Client{
		Transport: httpTransport,
	}

	return httpClient
}

func handleEventTypes(response string) {
	var CheckResult EventTypeCheckResult
	err := json.Unmarshal([]byte(response), &CheckResult)
	if err == nil {
//		fmt.Printf("%+v\n", CheckResult)
		//TODO: Add fall through for different types
	} else {
//		fmt.Println(err)
//		fmt.Printf("%+v\n", CheckResult)
	}

	if CheckResult.Type == "CheckResult" {
		//fmt.Println("Processing type 'CheckResult'")

		CheckResultCount++

		var ObjectName = CheckResult.Host
		var IsHost = true
		if CheckResult.Service != "" {
			ObjectName = ObjectName + "!" + CheckResult.Service
			IsHost = false
		}

		CheckResultCountObject[ObjectName] = CheckResultCountObject[ObjectName] + 1

		state, err := json.Number.Float64(CheckResult.CheckResult.State)
		if err != nil {
			fmt.Println("Cannot fetch state", CheckResult.CheckResult.State)
		}

		StateChangesObject[ObjectName] = append(StateChangesObject[ObjectName], int64(state))

		var StateCount = len(StateChangesObject[ObjectName])

		if StateCount > 10 {
			StateChangesObject[ObjectName] = StateChangesObject[ObjectName][StateCount-10:]
			StateCount = 10
		}

		fmt.Printf("Summary for object \x1b[1;1m%s\x1b[0m\n", ObjectName)
		fmt.Println("-----------------------------------")
		fmt.Printf(" State History: %s\n", displayStates(StateChangesObject[ObjectName][:], IsHost))
		fmt.Printf(" Output: '%s'\n", CheckResult.CheckResult.Output)

		if detectFlapping(StateChangesObject[ObjectName][:]) == true {
			fmt.Printf(" \x1b[31;1m===> ATTENTION: Object '%s' is flapping between states\x1b[0m: %s\n", ObjectName, displayStates(StateChangesObject[ObjectName][:], IsHost))
		}

		fmt.Println(" ")
		var CheckResultRate = float64(CheckResultCount) / float64(time.Now().Unix() - StartTime)
		fmt.Printf(" Global check result rate/second: %.02f\n", CheckResultRate)

		var CheckResultObjectRate = float64(CheckResultCountObject[ObjectName]) / float64(time.Now().Unix() - StartTime)
		fmt.Printf(" Check result rate for object '%s': %.02f\n", ObjectName, CheckResultObjectRate)
		fmt.Println(" ")
	}
}

type nameGenerator interface {
	randomName() string
}
type timeStampedNameGenerator struct{}

func (T timeStampedNameGenerator) randomName() string {
	return time.Now().Format(timeFormat)
}

type emptyNameGenerator struct{}

func (T emptyNameGenerator) randomName() string {
	return ""
}