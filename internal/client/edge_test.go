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

package client_test

import (
	"testing"

	"github.com/eclipse-kanto/system-metrics/internal/client"
)

func TestFetchConfigConnectError(t *testing.T) {
	cfg := &client.BrokerConfig{}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("failed to validate missing broker host: %v", err)
	}

	cfg.Broker = "tcp://unknownhost:1883"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("failed to validate broker host: %v", err)
	}

	_, err := client.NewEdgeManager(cfg, nil)
	if err == nil {
		t.Fatalf("failed to validate with unknown broker host: %v", err)
	}
}
