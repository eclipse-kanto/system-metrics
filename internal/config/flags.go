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

package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/eclipse-kanto/system-metrics/internal/client"
	"github.com/eclipse-kanto/system-metrics/internal/logger"
	"github.com/eclipse-kanto/system-metrics/internal/metrics"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

// MetricsConfig contains the Metrics binary configuration data.
type MetricsConfig struct {
	client.BrokerConfig
	metrics.Request
	logger.LogConfig
}

// Default returns the default metrics configuration.
func Default() *MetricsConfig {
	return &MetricsConfig{
		BrokerConfig: client.BrokerConfig{
			Broker: "tcp://localhost:1883",
		},
		LogConfig: logger.LogConfig{
			LogFile:       "log/system-metrics.log",
			LogLevel:      logger.INFO.String(),
			LogFileSize:   2,
			LogFileCount:  5,
			LogFileMaxAge: 28,
		},
	}
}

// Parse defines, reads config and parses all flags.
func Parse(f *flag.FlagSet, args []string, version string) (*MetricsConfig, error) {

	printVersion := f.Bool("version", false, "Prints current version and exits")
	configFile := f.String("configFile", "", "Configuration `file` in JSON format with flags values")

	add(f)

	if err := f.Parse(args); err != nil {
		return nil, err
	}

	if *printVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	metricsConfig := Default()
	if err := readConfig(*configFile, metricsConfig); err != nil {
		return nil, err
	}

	cli := copy(f)
	if err := mergo.Map(metricsConfig, cli, mergo.WithOverwriteWithEmptyValue); err != nil {
		return nil, errors.Wrap(err, "cannot process metrics")
	}

	if err := metricsConfig.validate(); err != nil {
		return nil, err
	}

	return metricsConfig, nil
}

func add(f *flag.FlagSet) *MetricsConfig {
	def := Default()

	metricsConfig := &MetricsConfig{}

	f.StringVar(&metricsConfig.Broker, "broker", def.Broker, "Local MQTT broker address")
	f.StringVar(&metricsConfig.Username, "username", def.Username, "Username for authorized local client")
	f.StringVar(&metricsConfig.Password, "password", def.Password, "Password for authorized local client")
	f.StringVar(&metricsConfig.CaCert, "caCert", def.CaCert, "A PEM encoded CA certificates `file` for MQTT broker connection")
	f.StringVar(&metricsConfig.ClientCert, "clientCert", def.ClientCert, "A PEM encoded certificate `file` to authenticate to the MQTT server/broker")
	f.StringVar(&metricsConfig.ClientKey, "clientKey", def.ClientKey, "A PEM encoded unencrypted private key `file` to authenticate to the MQTT server/broker")

	f.DurationVar(&metricsConfig.Frequency.Duration, "frequency", def.Frequency.Duration,
		"Initial frequency of publishing system data to cloud as duration string, e.g. 30s, 10m",
	)

	f.StringVar(&metricsConfig.LogFile, "logFile", def.LogFile, "Log file location in storage directory")
	f.StringVar(&metricsConfig.LogLevel, "logLevel", def.LogLevel, "Log levels are ERROR, WARNING, INFO, DEBUG, TRACE")
	f.IntVar(&metricsConfig.LogFileSize, "logFileSize", def.LogFileSize, "Log file size in MB before it gets rotated")
	f.IntVar(&metricsConfig.LogFileCount, "logFileCount", def.LogFileCount, "Log file max rotations count")
	f.IntVar(&metricsConfig.LogFileMaxAge, "logFileMaxAge", def.LogFileMaxAge, "Log file rotations max age in days")

	return metricsConfig
}

func readConfig(path string, config interface{}) error {
	if fileExists(path) {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "error reading configuration file: %s", path)
		}

		if len(data) == 0 {
			return nil
		}

		if err = json.Unmarshal(data, config); err != nil {
			return errors.Wrapf(err, "provided configuration file '%s' is not a valid JSON", path)
		}
	}

	return nil
}

func copy(f *flag.FlagSet) map[string]interface{} {
	m := make(map[string]interface{}, f.NFlag())

	f.Visit(func(f *flag.Flag) {
		name := f.Name
		getter := f.Value.(flag.Getter)
		v := getter.Get()

		if d, ok := v.(time.Duration); ok {
			v = metrics.Duration{
				Duration: d,
			}
		}

		m[name] = v
	})

	return m
}

func fileExists(filename string) bool {
	if len(filename) == 0 {
		return false
	}

	info, err := os.Stat(filename)
	if info == nil || os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (m *MetricsConfig) validate() error {
	if err := m.BrokerConfig.Validate(); err != nil {
		return err
	}

	if err := m.LogConfig.Validate(); err != nil {
		return err
	}

	return nil
}
