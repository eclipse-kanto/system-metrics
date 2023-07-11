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

package config_test

import (
	"flag"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/eclipse-kanto/system-metrics/internal/config"
	"github.com/eclipse-kanto/system-metrics/internal/logger"
	"github.com/eclipse-kanto/system-metrics/internal/metrics"
)

func TestFlagsParseNoConfig(t *testing.T) {
	args := []string{
		"-username=test",
		"-password=test",
		"-caCert=testCaCert",
		"-clientCert=clientCert",
		"-clientKey=clientKey",
		"-frequency=30s",
		"-logFile=P",
		"-logLevel=TRACE",
		"-logFileSize=10",
		"-logFileCount=100",
		"-logFileMaxAge=1000",
	}
	expConfig := config.Default()
	expConfig.Username = "test"
	expConfig.Password = "test"
	expConfig.CaCert = "testCaCert"
	expConfig.ClientCert = "clientCert"
	expConfig.ClientKey = "clientKey"
	expConfig.Frequency.Duration = 30 * time.Second
	expConfig.LogFile = "P"
	expConfig.LogLevel = logger.TRACE.String()
	expConfig.LogFileSize = 10
	expConfig.LogFileCount = 100
	expConfig.LogFileMaxAge = 1000
	f := flag.NewFlagSet("testing", flag.ContinueOnError)

	parsed, err := config.Parse(f, args, "0.0.0")

	if err != nil {
		t.Errorf("Failed parse metrics arguments, err: %v", err)
	}
	assertConfig(t, "TestFlagsParseNoConfig", expConfig, parsed)
}

func TestConfigParse(t *testing.T) {
	args := []string{
		"-password=test",
		"-configFile=testdata/config.json",
	}
	expConfig := config.Default()
	expConfig.Username = "Username_config"
	expConfig.Password = "test"
	expConfig.CaCert = "CaCert_config"
	expConfig.ClientCert = "Cert_config"
	expConfig.ClientKey = "Key_config"

	expConfig.Frequency.Duration = 5 * time.Minute
	expConfig.Filter = []metrics.Filter{
		{
			Originator: "SYSTEM",
		},
		{
			ID:         []string{"io.*", "cpu.*", "memory.*"},
			Originator: "test.process",
		},
	}

	expConfig.LogFile = "logFile_config"
	expConfig.LogLevel = logger.DEBUG.String()

	f := flag.NewFlagSet("testing", flag.ContinueOnError)

	parsed, err := config.Parse(f, args, "0.0.0")

	if err != nil {
		t.Errorf("Failed parse metrics arguments, err: %v", err)
	}
	assertConfig(t, "TestConfigParse", expConfig, parsed)
}

func TestFlagsOverrideConfig(t *testing.T) {
	args := []string{
		"-username=test",
		"-password=test",
		"-caCert=testCaCert",
		"-clientCert=clientCert",
		"-clientKey=clientKey",
		"-frequency=30m",
		"-logFile=P",
		"-logLevel=TRACE",
		"-logFileSize=10",
		"-logFileCount=100",
		"-logFileMaxAge=1000",
		"-configFile=testdata/config.json",
	}
	expConfig := config.Default()
	expConfig.Username = "test"
	expConfig.Password = "test"
	expConfig.CaCert = "testCaCert"
	expConfig.ClientCert = "clientCert"
	expConfig.ClientKey = "clientKey"
	expConfig.Frequency.Duration = 30 * time.Minute
	expConfig.Filter = []metrics.Filter{
		{
			Originator: "SYSTEM",
		},
		{
			ID:         []string{"io.*", "cpu.*", "memory.*"},
			Originator: "test.process",
		},
	}
	expConfig.LogFile = "P"
	expConfig.LogLevel = logger.TRACE.String()
	expConfig.LogFileSize = 10
	expConfig.LogFileCount = 100
	expConfig.LogFileMaxAge = 1000
	f := flag.NewFlagSet("testing", flag.ContinueOnError)

	parsed, err := config.Parse(f, args, "0.0.0")

	if err != nil {
		t.Errorf("Failed parse metrics arguments, err: %v", err)
	}
	assertConfig(t, "TestFlagsOverrideConfig", expConfig, parsed)
}

func TestConfigNonexisting(t *testing.T) {
	args := []string{
		"-configFile=invalid_config.json",
	}

	f := flag.NewFlagSet("testing", flag.ContinueOnError)
	parsed, err := config.Parse(f, args, "0.0.0")

	if err != nil {
		t.Errorf("Failed parse metrics arguments, err: %v", err)
	}
	assertConfig(t, "TestConfigNonexisting", config.Default(), parsed)
}

func TestInvalidFlagType(t *testing.T) {
	args := []string{
		"-version=1.2.3",
	}
	f := flag.NewFlagSet("testing", flag.ContinueOnError)

	_, err := config.Parse(f, args, "0.0.0")
	if !assertEqualsErrors(err, "invalid boolean value \"1.2.3\" for -version: parse error") {
		t.Errorf(
			"Expected error 'invalid boolean value \"1.2.3\" for -version: parse error', but received err: %v", err)
	}
}

func TestValidateBrokerConfigInvalid(t *testing.T) {
	f := flag.NewFlagSet("testing", flag.ContinueOnError)
	args := []string{
		"-broker=",
	}

	_, err := config.Parse(f, args, "0.0.0")
	if !assertEqualsErrors(err, "broker is missing") {
		t.Errorf("Expected error 'broker is missing', but received  err: %v", err)
	}
}

func TestValidateBrokerConfigInvalidMQTTTLSflags(t *testing.T) {
	f := flag.NewFlagSet("testing", flag.ContinueOnError)
	args := []string{
		"-broker=test",
		"-caCert=testCaCert",
		"-clientCert=clientCert",
	}

	_, err := config.Parse(f, args, "0.0.0")
	if !assertEqualsErrors(err, "either both client MQTT certificate and key must be set or none of them") {
		t.Errorf("Expected error 'either both client MQTT certificate and key must be set or none of them', but received  err: %v", err)
	}
}

func TestValidateLogConfigInvalidFileSize(t *testing.T) {
	f := flag.NewFlagSet("testing", flag.ContinueOnError)
	args := []string{
		"-logFileSize=-1",
	}

	_, err := config.Parse(f, args, "0.0.0")
	if !assertEqualsErrors(err, "logFileSize <= 0") {
		t.Errorf("Expected error 'logFileSize <= 0', but received  err: %v", err)
	}
}

func TestValidateLogConfigInvalidFileCount(t *testing.T) {
	f := flag.NewFlagSet("testing", flag.ContinueOnError)
	args := []string{
		"-logFileCount=-5",
	}

	_, err := config.Parse(f, args, "0.0.0")
	if !assertEqualsErrors(err, "logFileCount <= 0") {
		t.Errorf("Expected error 'logFileCount <= 0', but received  err: %v", err)
	}
}

func TestValidateLogConfigInvalidFileMaxAge(t *testing.T) {
	f := flag.NewFlagSet("testing", flag.ContinueOnError)
	args := []string{
		"-logFileMaxAge=-10",
	}

	_, err := config.Parse(f, args, "0.0.0")
	if !assertEqualsErrors(err, "logFileMaxAge < 0") {
		t.Errorf("Expected error 'logFileMaxAge < 0', but received err: %v", err)
	}
}

func assertConfig(t *testing.T, test string, exp, actual *config.MetricsConfig) {
	if !reflect.DeepEqual(&exp.BrokerConfig, &actual.BrokerConfig) {
		t.Errorf("[%s] Broker config does not match: %v != %v", test, exp.BrokerConfig, actual.BrokerConfig)
	}

	if !reflect.DeepEqual(&exp.Request, &actual.Request) {
		t.Errorf("[%s] Request config does not match: %v != %v", test, exp.Request, actual.Request)
	}

	if !reflect.DeepEqual(&exp.LogConfig, &actual.LogConfig) {
		t.Errorf("[%s] Log config does not match: %v != %v", test, exp.LogConfig, actual.LogConfig)
	}
}

func assertEqualsErrors(err error, wantMsg string) bool {
	if err == nil {
		return wantMsg == ""
	}
	if wantMsg == "" {
		return false
	}
	return strings.Contains(err.Error(), wantMsg)
}
