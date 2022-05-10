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

package metrics

import (
	"testing"
	"time"
)

func TestMetricsRequestHasFilterFor(t *testing.T) {
	type fields struct {
		Filters []Filter
	}
	tests := []struct {
		name       string
		fields     fields
		originator string
		want       bool
	}{
		{
			name: "testOriginatorSystem",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{},
					Originator: "SYSTEM",
				}},
			},
			originator: "SYSTEM",
			want:       true,
		},
		{
			name: "testOriginatorEmpty",
			fields: fields{
				Filters: []Filter{{
					ID: []string{"test"},
				}},
			},
			originator: "SYSTEM",
			want:       true,
		},
		{
			name:       "testFiltersNone",
			fields:     fields{},
			originator: "SYSTEM",
			want:       true,
		},
		{
			name: "testOriginatorNonMatching",
			fields: fields{
				Filters: []Filter{{
					ID:         nil,
					Originator: "unknown",
				}},
			},
			originator: "SYSTEM",
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &Request{
				Frequency: Duration{5 * time.Second},
				Filter:    tt.fields.Filters,
			}
			mr.normalize()
			if got := mr.hasFilterFor(tt.originator); got != tt.want {
				t.Errorf("MetricsRequest.HasFilterFor = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricsRequestHasFilterForItem(t *testing.T) {
	type fields struct {
		Filters []Filter
	}
	type input struct {
		Originator string
		ID         string
	}
	tests := []struct {
		name   string
		fields fields
		arg    input
		want   bool
	}{
		{
			name: "testWithSame",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.utilization"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testMultipleWithSame",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.utilization", "memory.utilization"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithEmptyOriginator",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.utilization", "memory.utilization"},
					Originator: "",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithEmptyID",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{},
					Originator: "",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithNoID",
			fields: fields{
				Filters: []Filter{{
					Originator: "",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithWildcardExact",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.*"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithWildcardGroup",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"memory.*"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         memoryGroupPrefix,
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithWildcardProcess",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"io.*"},
					Originator: "test",
				}},
			},
			arg: input{
				ID:         ioGroupPrefix,
				Originator: "test",
			},
			want: true,
		},
		{
			name: "testWithWildcardFail",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.*"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpuA.utilization",
				Originator: "SYSTEM",
			},
			want: false,
		},
		{
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.unknown", "memory.unknown"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: false,
		},
		{
			name: "testFalseOriginator",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.unknown"},
					Originator: "unknown",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &Request{
				Frequency: Duration{5 * time.Minute},
				Filter:    tt.fields.Filters,
			}
			mr.normalize()
			if got := mr.hasFilterForItem(tt.arg.ID, tt.arg.Originator); got != tt.want {
				t.Errorf("MetricsRequest.HasFilterForItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricsRequestHasFilterForGroup(t *testing.T) {
	type fields struct {
		Filters []Filter
	}
	type input struct {
		Originator string
		ID         string
	}
	tests := []struct {
		name   string
		fields fields
		arg    input
		want   bool
	}{
		{
			name: "testWithSame",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.utilization"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpu.utilization",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testMultipleWithSame",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.utilization", "memory.utilization"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         memoryGroupPrefix,
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testCPULoadGroup",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"memory.utilization", "cpu.load15"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         cpuLoadGroupPrefix,
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithEmptyID",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{},
					Originator: "",
				}},
			},
			arg: input{
				ID:         memoryGroupPrefix,
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithWildcard",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"io.*"},
					Originator: "process",
				}},
			},
			arg: input{
				ID:         ioGroupPrefix,
				Originator: "process",
			},
			want: true,
		},
		{
			name: "testWithSame",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.load1"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpu.load1",
				Originator: "SYSTEM",
			},
			want: true,
		},
		{
			name: "testWithWildcardFail",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"cpu.*"},
					Originator: "SYSTEM",
				}},
			},
			arg: input{
				ID:         "cpuA.",
				Originator: "SYSTEM",
			},
			want: false,
		},
		{
			name: "testFalseOriginator",
			fields: fields{
				Filters: []Filter{{
					ID:         []string{"memory.total"},
					Originator: "unknown",
				}},
			},
			arg: input{
				ID:         memoryGroupPrefix,
				Originator: "SYSTEM",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &Request{
				Frequency: Duration{5 * time.Minute},
				Filter:    tt.fields.Filters,
			}
			mr.normalize()
			if got := mr.hasFilterForGroups(tt.arg.Originator, tt.arg.ID); got != tt.want {
				t.Errorf("MetricsRequest.HasFilterForGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
