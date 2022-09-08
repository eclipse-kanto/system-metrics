// Copyright (c) 2022 Contributors to the Eclipse Foundation
//
// See the NOTICE file(s) distributed with this work for additional
// information regarding copyright ownership.
//
// This program and the accompanying materials are made available under the
// terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0, or the Apache License, Version 2.0
// which is available at https://www.apache.org/licenses/LICENSE-2.0.
//
// SPDX-License-Identifier: EPL-2.0 OR Apache-2.0

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eclipse-kanto/system-metrics/internal/client"
	"github.com/eclipse-kanto/system-metrics/internal/config"
	"github.com/eclipse-kanto/system-metrics/internal/logger"
	"github.com/eclipse-kanto/system-metrics/internal/metrics"
)

const (
	exitOk  = 0
	exitErr = 1
)

var (
	version = "dev"
)

// initialize the ditto client (with connectHandler and messageHandler),
// create edge connector, responsible for connecting and disconnecting the client and listening for thing info changes
// and wait for program exit signal
func run(cfg *config.MetricsConfig) int {
	loggerOut := logger.SetupLogger(&cfg.LogConfig)
	defer loggerOut.Close()

	logger.Infof("Starting System Metrics %v", version)
	defer logger.Info("Exiting...")

	chExit := make(chan os.Signal, 1)
	signal.Notify(chExit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(chExit)

	metricsFeature := metrics.NewMetrics(&cfg.Request)

	edgeManager, err := client.NewEdgeManager(&cfg.BrokerConfig, metricsFeature)
	if err != nil {
		logger.Errorf("Error creating edge connector: %v", err)
		return exitErr
	}

	defer edgeManager.Close()

	<-chExit

	return exitOk
}

func main() {
	f := flag.NewFlagSet("metrics", flag.ExitOnError)

	cfg, err := config.Parse(f, os.Args[1:], version)
	if err != nil {
		log.Fatalf("Could not parse arguments: %v", err)
	}

	os.Exit(run(cfg))
}
