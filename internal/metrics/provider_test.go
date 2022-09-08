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
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/eclipse-kanto/system-metrics/internal/logger"
)

func TestProcessPIDOf(t *testing.T) {
	tests := []struct {
		name    string
		process string
		want    int32
		wantErr bool
	}{
		{
			name:    "testProccessNotFound",
			process: "invalid",
			want:    1,
			wantErr: true,
		},
		{
			name:    "testProccessUnix",
			process: "systemd",
			want:    0,
			wantErr: runtime.GOOS != "linux",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processPIDOf(tt.process)
			if (err != nil) != tt.wantErr {
				t.Errorf("processPIDOf error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if got == tt.want {
					t.Errorf("processPIDOf got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestHandleRequest(t *testing.T) {
	tests := []struct {
		name              string
		request           *Request
		process           string
		withCorrelationID bool
		expDataIDs        int
		expErr            bool
	}{
		{
			name: "testSystemMetricsRequest",
			request: &Request{
				Frequency: Duration{2 * time.Second},
				Filter: []Filter{{
					ID:         []string{cpuUtilization, memoryUtilization},
					Originator: originatorSystem,
				}},
			},
			process:           originatorSystem,
			withCorrelationID: true,
			expDataIDs:        2,
			expErr:            false,
		},
		{
			name: "testSystemMetricsRequestNoCorrelationID",
			request: &Request{
				Frequency: Duration{2 * time.Second},
				Filter: []Filter{{
					ID:         []string{"memory.*", "cpu.*"},
					Originator: originatorSystem,
				}},
			},
			process:    originatorSystem,
			expDataIDs: 8,
			expErr:     false,
		},
		{
			name: "testProcessMetricsRequest",
			request: &Request{
				Frequency: Duration{3 * time.Second},
				Filter: []Filter{{
					ID:         []string{"io.*", "cpu.*", "memory.*"},
					Originator: testingProcess(),
				}},
			},
			process:    testingProcess(),
			expDataIDs: 4,
			expErr:     false,
		},
	}

	provider := testProvider()
	defer provider.cancelProcess()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.process) == 0 {
				t.Skipf("no permission to run '%s'", tt.name)
			}

			if tt.withCorrelationID {
				tt.request.correlationID = tt.name
			}
			err := provider.handleRequest(tt.request)

			if err != nil {
				if !tt.expErr {
					t.Errorf("handleRequest for '%v'[%s], error = %v, wantErr %v",
						tt.request, tt.process, err, tt.expErr)
				}

			} else {
				received := <-provider.consumer.rData
				if received.Timestamp == 0 {
					t.Errorf("%s[%s] response does not containt timestamp",
						tt.name,
						tt.process,
					)
				}
				if len(received.Snapshot) != 1 {
					t.Errorf("%s[%s] did not retrieve expected originator metrics, received data '%v'",
						tt.name,
						tt.process,
						received,
					)
				}
				originatorMeasurements := received.Snapshot[0]
				if originatorMeasurements.Originator != tt.process {
					t.Errorf("%s[%s] originator mismatch, received data '%v'",
						tt.name,
						tt.process,
						originatorMeasurements.Originator,
					)
				}
				if len(originatorMeasurements.Measurements) != tt.expDataIDs {
					t.Errorf("%s[%s] does not retrieve expected data count, expected %d, received data '%v'",
						tt.name,
						tt.process,
						tt.expDataIDs,
						originatorMeasurements.Measurements,
					)
				}
				if provider.consumer.rID != tt.request.correlationID {
					t.Errorf("%s does not contain expected correlationID, expected '%s', received data '%s'",
						tt.name,
						tt.request.correlationID,
						provider.consumer.rID,
					)
				}
			}
		})
	}
}

func testingProcess() string {
	return filepath.Base(os.Args[0])
}

type TestProvider struct {
	provider
	consumer *TestConsumer
}

func testProvider() *TestProvider {
	logger.SetupLogger(&logger.LogConfig{LogLevel: "TRACE"})

	tConsumer := TestConsumer{}
	tConsumer.rData = make(chan MetricData, 1)
	return &TestProvider{
		consumer: &tConsumer,
		provider: provider{
			consumer: tConsumer.consume,
		},
	}
}

type TestConsumer struct {
	rID   string
	rData chan MetricData
}

func (c *TestConsumer) consume(m MetricData, ID string) {
	c.rID = ID
	c.rData <- m
}
