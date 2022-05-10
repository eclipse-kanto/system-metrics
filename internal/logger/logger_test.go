// Copyright (c) 2022 Contributors to the Eclipse Foundation
//
// See the NOTICE file(s) distributed with this work for additional
// information regarding copyright ownership.
//
// This program and the accompanying materials are made available under the
// terms of the Eclipse Public License 2.0 which is available at
// http://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package logger_test

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eclipse-kanto/system-metrics/internal/logger"
)

const (
	logLocation = "_test-logger"
)

// TestLogLevelError tests logger functions with log level set to ERROR.
func TestLogLevelError(t *testing.T) {
	validate(t, logger.ERROR, "TestLogLevelError")
}

// TestLogLevelWarning tests logger functions with log level set to WARNING.
func TestLogLevelWarning(t *testing.T) {
	validate(t, logger.WARN, "TestLogLevelWarning")
}

// TestLogLevelInfo tests logger functions with log level set to INFO.
func TestLogLevelInfo(t *testing.T) {
	validate(t, logger.INFO, "TestLogLevelInfo")
}

// TestLogLevelDebug tests logger functions with log level set to DEBUG.
func TestLogLevelDebug(t *testing.T) {
	validate(t, logger.DEBUG, "TestLogLevelDebug")
}

// TestLogLevelTrace tests logger functions with log level set to TRACE.
func TestLogLevelTrace(t *testing.T) {
	validate(t, logger.TRACE, "TestLogLevelTrace")
}

// TestNopWriter tests logger functions without writter.
func TestNopWriter(t *testing.T) {
	createLogDir(t)
	defer os.RemoveAll(logLocation)

	// Prepare the logger without writter
	loggerOut := logger.SetupLogger(&logger.LogConfig{
		LogFile: "", LogLevel: logger.TRACE.String(), LogFileSize: 2, LogFileCount: 5,
	})
	defer loggerOut.Close()

	logger.Error("test error", errors.New("simple error"))
	f, err := os.Open(logLocation)
	if err != nil {
		t.Fatalf("cannot open test directory: %v", err)
	}
	defer f.Close()

	if _, err = f.Readdirnames(1); err != io.EOF {
		t.Errorf("test directory is not empty")
	}
}

func createLogDir(t *testing.T) {
	if err := os.MkdirAll(logLocation, 0755); err != nil {
		t.Fatalf("failed create test directory: %v", err)
	}
}

func validate(t *testing.T, logLevel logger.LogLevel, testName string) {
	createLogDir(t)
	defer os.RemoveAll(logLocation)

	logPath := filepath.Join(logLocation, logLevel.String()+".log")
	loggerOut := logger.SetupLogger(&logger.LogConfig{
		LogFile: logPath, LogLevel: logLevel.String(), LogFileSize: 2, LogFileCount: 5,
	})
	defer loggerOut.Close()

	validateError(t, logPath, logLevel, testName)
	validateWarning(t, logPath, logLevel, testName)
	validateInfo(t, logPath, logLevel, testName)
	validateDebug(t, logPath, logLevel, testName)
	validateTrace(t, logPath, logLevel, testName)
}

func validateError(t *testing.T, log string, logLevel logger.LogLevel, testName string) {
	logExpected := logLevel.Enabled(logger.ERROR)

	logger.Error("error log", errors.New("simple error"))
	if logExpected != search(t, log, logger.ERROR.StringAligned(), "error log, err: simple error") {
		t.Errorf("[%s] error entry mismatch [result: %v]", testName, !logExpected)
	}

	logger.Error("error log without error", nil)
	if logExpected != search(t, log, logger.ERROR.StringAligned(), "error log without error") {
		t.Errorf("[%s] error entry without error mismatch [result: %v]", testName, !logExpected)
	}

	logger.Errorf("error log [%v,%s]", "param1", "param2")
	if logExpected != search(t, log, logger.ERROR.StringAligned(), "error log [param1,param2]") {
		t.Errorf("[%s] errorf entry mismatch: [result: %v]", testName, !logExpected)
	}
}

func validateWarning(t *testing.T, log string, logLevel logger.LogLevel, testName string) {
	logExpected := logLevel.Enabled(logger.WARN)

	logger.Warning("warning log", errors.New("simple warning error"))
	if logExpected != search(t, log, logger.WARN.StringAligned(), "warning log, err: simple warning error") {
		t.Errorf("[%s] warning entry mismatch [result: %v]", testName, !logExpected)
	}

	logger.Warningf("warning log [%v,%s]", "param1", "param2")
	if logExpected != search(t, log, logger.WARN.StringAligned(), "warning log [param1,param2]") {
		t.Errorf("[%s] warningf entry mismatch: [result: %v]", testName, !logExpected)
	}
}

func validateInfo(t *testing.T, log string, logLevel logger.LogLevel, testName string) {
	logExpected := logLevel.Enabled(logger.INFO)

	logger.Info("info log")
	if logExpected != search(t, log, logger.INFO.StringAligned(), "info log") {
		t.Errorf("[%s] info entry mismatch [result: %v]", testName, !logExpected)
	}

	logger.Infof("info log [%v,%s]", "param1", "param2")
	if logExpected != search(t, log, logger.INFO.StringAligned(), "info log [param1,param2]") {
		t.Errorf("[%s] infof entry mismatch: [result: %v]", testName, !logExpected)
	}
}

func validateDebug(t *testing.T, log string, logLevel logger.LogLevel, testName string) {
	logExpected := logLevel.Enabled(logger.DEBUG)

	logger.Debug("debug log")
	if logExpected != search(t, log, logger.DEBUG.StringAligned(), "debug log") {
		t.Errorf("[%s] debug entry mismatch [result: %v]", testName, !logExpected)
	}

	logger.Debugf("debug log [%v,%s]", "param1", "param2")
	if logExpected != search(t, log, logger.DEBUG.StringAligned(), "debug log [param1,param2]") {
		t.Errorf("[%s] debugf entry mismatch: [result: %v]", testName, !logExpected)
	}
}

func validateTrace(t *testing.T, log string, logLevel logger.LogLevel, testName string) {
	logExpected := logLevel.Enabled(logger.TRACE)

	logger.Trace("trace log")
	if logExpected != search(t, log, logger.TRACE.StringAligned(), "trace log") {
		t.Errorf("[%s] trace entry mismatch [result: %v]", testName, !logExpected)
	}

	logger.Tracef("trace log [%v,%s]", "param1", "param2")
	if logExpected != search(t, log, logger.TRACE.StringAligned(), "trace log [param1,param2]") {
		t.Errorf("[%s] tracef entry mismatch: [result: %v]", testName, !logExpected)
	}
}

func search(t *testing.T, fn string, entries ...string) bool {
	file, err := os.Open(fn)
	if err != nil {
		t.Fatalf("fail to open log file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if contains(scanner.Text(), entries...) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("fail to read log file: %v", err)
	}
	return false
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}
