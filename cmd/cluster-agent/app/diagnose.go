// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package app

import (
	"fmt"

	"github.com/DataDog/datadog-agent/cmd/agent/common"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/diagnose"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/apiserver"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var diagnoseNames = []string{
	apiserver.DiagnoseName,
}

func init() {
	ClusterAgentCmd.AddCommand(diagnoseCommand)
}

var diagnoseCommand = &cobra.Command{
	Use:   "diagnose",
	Short: "Execute some connectivity diagnosis on your system",
	Long:  ``,
	Run:   doDiagnose,
}

func doDiagnose(cmd *cobra.Command, args []string) {
	// Global config setup
	if confPath != "" {
		if err := common.SetupConfig(confPath); err != nil {
			fmt.Println("Cannot setup config, exiting:", err)
			panic(err)
		}
	}

	if flagNoColor {
		color.NoColor = true
	}

	err := config.SetupLogger(
		loggerName,
		config.Datadog.GetString("log_level"),
		common.DefaultLogFile,
		config.GetSyslogURI(),
		config.Datadog.GetBool("syslog_rfc"),
		config.Datadog.GetBool("log_to_console"),
		config.Datadog.GetBool("log_format_json"),
	)
	if err != nil {
		log.Errorf("Error while setting up logging, exiting: %v", err)
		panic(err)
	}

	err = diagnose.Run(color.Output, diagnoseNames)
	if err != nil {
		panic(err)
	}
}
