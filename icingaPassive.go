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
	"strings"
	"time"

	"github.com/getgauge/common"
	"github.com/guandalf/icinga-passive/builder"
	"github.com/guandalf/icinga-passive/gauge_messages"
	"github.com/guandalf/icinga-passive/listener"
)

const (
	EXECUTION_ACTION            = "execution"
	GAUGE_HOST                  = "127.0.0.1"
	GAUGE_PORT_ENV              = "plugin_connection_port"
	PLUGIN_ACTION_ENV           = "icinga-psv-notify_action"
	icingaPassive               = "icinga-psv-notify"
	overwriteReportsEnvProperty = "overwrite_reports"
	icingaHostname              = "icinga_hostname"
	timeFormat                  = "2006-01-02 15.04.05"
)

var projectRoot string
var pluginDir string

func createReport(suiteResult *gauge_messages.SuiteExecutionResult) {
	bytes, err := builder.NewXmlBuilder(0).GetXmlContent(suiteResult)
	if err != nil {
		fmt.Printf("Report generation failed: %s \n", err)
		os.Exit(1)
	}
	fmt.Printf("Sucessfully run icinga-passive %s\n", bytes)
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

func getNameGen() nameGenerator {
	if shouldOverwriteReports() {
		return emptyNameGenerator{}
	}
	return timeStampedNameGenerator{}
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

func shouldOverwriteReports() bool {
	envValue := os.Getenv(overwriteReportsEnvProperty)
	if strings.ToLower(envValue) == "true" {
		return true
	}
	return false
}
