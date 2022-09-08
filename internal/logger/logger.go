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

package logger

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig represents a log configuration.
type LogConfig struct {
	LogFile       string
	LogLevel      string
	LogFileSize   int
	LogFileCount  int
	LogFileMaxAge int
}

const (
	prefix = "[metrics]"

	logFlags int = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lmsgprefix
)

var (
	logger *log.Logger
	level  LogLevel
)

// SetupLogger initializes the log based on the provided log configuration.
func SetupLogger(logConfig *LogConfig) io.WriteCloser {
	loggerOut := newLoggerOut(logConfig)

	log.SetOutput(loggerOut)
	log.SetFlags(logFlags)

	format := fmt.Sprintf(" %%-%vs", len(prefix)+2)
	logger = log.New(loggerOut, fmt.Sprintf(format, prefix), logFlags)
	level = ParseLogLevel(logConfig.LogLevel)

	return loggerOut
}

func newLoggerOut(logConfig *LogConfig) io.WriteCloser {
	if len(logConfig.LogFile) != 0 {
		if err := os.MkdirAll(filepath.Dir(logConfig.LogFile), 0755); err == nil {
			return &lumberjack.Logger{
				Filename:   logConfig.LogFile,
				MaxSize:    logConfig.LogFileSize,
				MaxBackups: logConfig.LogFileCount,
				MaxAge:     logConfig.LogFileMaxAge,
				LocalTime:  true,
				Compress:   true,
			}
		}

	}
	return io.WriteCloser(&nopWriterCloser{out: os.Stderr})
}

// Validate validates the Logger settings.
func (c *LogConfig) Validate() error {
	if c.LogFileSize <= 0 {
		return errors.New("logFileSize <= 0")
	}

	if c.LogFileCount <= 0 {
		return errors.New("logFileCount <= 0")
	}

	if c.LogFileMaxAge < 0 {
		return errors.New("logFileMaxAge < 0")
	}

	return nil
}

// Error writes an error entry to the log with given message and error.
func Error(msg string, err error) {
	if level.Enabled(ERROR) {
		logger.Println(ERROR.StringAligned(), createErrorMsg(msg, err))
	}
}

// Errorf formats according to a format specifier and writes the string as an
// error entry to the log.
func Errorf(format string, a ...interface{}) {
	if level.Enabled(ERROR) {
		logger.Println(ERROR.StringAligned(), fmt.Sprintf(format, a...))
	}
}

// Warning writes a warning entry to the log with given message and error.
func Warning(msg string, err error) {
	if level.Enabled(WARN) {
		logger.Println(WARN.StringAligned(), createErrorMsg(msg, err))
	}
}

// Warningf formats according to a format specifier and writes the string as a
// warning entry to the log.
func Warningf(format string, a ...interface{}) {
	if level.Enabled(WARN) {
		logger.Println(WARN.StringAligned(), fmt.Sprintf(format, a...))
	}
}

// Info writes an info entry to the log.
func Info(msg string) {
	if level.Enabled(INFO) {
		logger.Println(INFO.StringAligned(), msg)

	}
}

// Infof formats according to a format specifier and writes the string as an
// info entry to the log.
func Infof(format string, a ...interface{}) {
	if level.Enabled(INFO) {
		logger.Println(INFO.StringAligned(), fmt.Sprintf(format, a...))
	}
}

// Debug writes a debug entry to the log.
func Debug(msg string) {
	if IsDebugEnabled() {
		logger.Println(DEBUG.StringAligned(), msg)
	}
}

// Debugf formats according to a format specifier and writes the string as a
// debug entry to the log.
func Debugf(format string, a ...interface{}) {
	if IsDebugEnabled() {
		logger.Println(DEBUG.StringAligned(), fmt.Sprintf(format, a...))
	}
}

// Trace writes a trace entry to the log.
func Trace(msg string) {
	if IsTraceEnabled() {
		logger.Println(TRACE.StringAligned(), msg)
	}
}

// Tracef formats according to a format specifier and write the string as a
// trace entry to the log.
func Tracef(format string, a ...interface{}) {
	if IsTraceEnabled() {
		logger.Println(TRACE.StringAligned(), fmt.Sprintf(format, a...))
	}
}

// IsDebugEnabled checks if debug log level is enabled.
func IsDebugEnabled() bool {
	return level.Enabled(DEBUG)
}

// IsTraceEnabled checks if trace log level is enabled.
func IsTraceEnabled() bool {
	return level.Enabled(TRACE)
}

func createErrorMsg(msg string, err error) string {
	if err != nil {
		return fmt.Sprintf("%s, err: %v", msg, err)
	}
	return msg
}

type nopWriterCloser struct {
	out io.Writer
}

func (w *nopWriterCloser) Write(p []byte) (n int, err error) {
	return w.out.Write(p)
}

func (*nopWriterCloser) Close() error {
	return nil
}
