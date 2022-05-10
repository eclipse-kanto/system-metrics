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

package logger

import (
	"strings"
)

// LogLevel defines the logger levels.
type LogLevel int

const (
	// ERROR is used to define the errors logging level.
	ERROR LogLevel = 1 + iota
	// WARN is used to define the errors and warnings logging level.
	WARN
	// INFO is used to define the errors, warnings and information logging level.
	INFO
	// DEBUG is used to define the errors, warnings, information and debug logging level.
	DEBUG
	// TRACE is used to define the more detailed logging level.
	TRACE
)

// Enabled returns true if the level is enabled.
func (l LogLevel) Enabled(level LogLevel) bool {
	return l >= level
}

func (l LogLevel) String() string {
	switch l {
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	case TRACE:
		return "TRACE"
	default:
		return "ERROR"
	}
}

// StringAligned returns the log level as string with padding.
func (l LogLevel) StringAligned() string {
	switch l {
	case WARN:
		return "WARN  "
	case INFO:
		return "INFO  "
	case DEBUG:
		return "DEBUG "
	case TRACE:
		return "TRACE "
	default:
		return "ERROR "
	}
}

// ParseLogLevel returns a LogLevel for a level string representation.
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "WARN":
		return WARN
	case "INFO":
		return INFO
	case "DEBUG":
		return DEBUG
	case "TRACE":
		return TRACE
	default:
		return ERROR
	}
}
