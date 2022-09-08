// Copyright (c) 2022 Contributors to the Eclipse Foundation
//
// See the NOTICE file(s) distributed with this work for additional
// information regarding copyright ownership.
//
// This program and the accompanying materials are made available under the
// terms of the Eclipse Public License v. 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0, or the Apache License, Version 2.0
// which is available at https://www.apache.org/licenses/LICENSE-2.0.
//
// SPDX-License-Identifier: EPL-2.0 OR Apache-2.0

package metrics

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	cpuUtilization = "cpu.utilization"

	cpuLoadGroupPrefix = "cpu.load"
	cpuLoad1           = "cpu.load1"
	cpuLoad5           = "cpu.load5"
	cpuLoad15          = "cpu.load15"

	memoryGroupPrefix = "memory."
	memoryTotal       = "memory.total"
	memoryAvailable   = "memory.available"
	memoryUsed        = "memory.used"
	memoryUtilization = "memory.utilization"

	ioGroupPrefix = "io."
	ioReadBytes   = "io.readBytes"
	ioWriteBytes  = "io.writeBytes"

	originatorSystem = "SYSTEM"
)

// Request defines the metric data request with defined frequency.
type Request struct {
	Frequency     Duration
	Filter        []Filter
	correlationID string
}

// Filter defines the type of metric data to be reported.
type Filter struct {
	ID         []string `json:"id"`
	Originator string   `json:"originator"`
}

// Duration is used to support duration string un-marshalling to time.Duration.
type Duration struct {
	time.Duration
}

// UnmarshalJSON supports '50s' string format.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value) * time.Second
		return nil

	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil

	default:
		return errors.Errorf("invalid duration: %v", v)
	}
}

// MarshalJSON supports marshalling to '50s' string format.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (mr *Request) normalize() {
	if mr.Frequency.Duration <= 0 {
		mr.Filter = nil

	} else if len(mr.Filter) == 0 {
		mr.Filter = []Filter{
			{
				ID:         []string{cpuUtilization, memoryUtilization},
				Originator: originatorSystem,
			},
		}

	} else {
		for i, f := range mr.Filter {
			if len(f.Originator) == 0 {
				mr.Filter[i] = Filter{
					ID:         f.ID,
					Originator: originatorSystem,
				}
			}
		}
	}
}

// hasFilterFor returns true if there is filter for the provided originator.
func (mr *Request) hasFilterFor(originator string) bool {
	for _, f := range mr.Filter {
		if f.Originator == originator {
			return true
		}
	}
	return false
}

// hasFilterForItem returns true if there is filter for same originator and filter's ID
// that is the same or with wildcard for the last ID segment,
// i.e. for provided "cpu.utilization" will return true if there is "cpu.utilization", "cpu.*" filter ID or if no filters are set.
func (mr *Request) hasFilterForItem(dataID, dataOriginator string) bool {
	for _, f := range mr.Filter {
		if f.Originator == dataOriginator {
			if len(f.ID) == 0 {
				return true
			}
			for _, fid := range f.ID {
				if fid == dataID {
					return true
				} else if strings.HasPrefix(dataID, strings.Trim(fid, "*")) {
					return true
				}
			}
		}
	}
	return false
}

// hasFilterForGroups returns true if there is filter for same originator and filter's ID
// that is the same or with same group prefix,
// i.e. for provided "cpu.load" group will return true if there is "cpu.load1", "cpu.*" filter ID or if no filters are set.
func (mr *Request) hasFilterForGroups(dataOriginator string, groupIDs ...string) bool {
	for _, f := range mr.Filter {
		if f.Originator == dataOriginator {
			if len(f.ID) == 0 {
				return true
			}
			for _, fid := range f.ID {
				for _, groupID := range groupIDs {
					if strings.HasPrefix(fid, groupID) {
						return true
					}
				}
			}
		}
	}
	return false
}
