// Copyright (c) 2021 Contributors to the Eclipse Foundation
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

//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/eclipse-kanto/kanto/integration/util"
	"github.com/eclipse-kanto/system-metrics/internal/metrics"

	"github.com/eclipse/ditto-clients-golang/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	metricsFeatureID = "Metrics"
)

type SystemMetricsSuite struct {
	suite.Suite

	util.SuiteInitializer

	metricsThingURL   string
	metricsFeatureURL string

	topicData string
}

func (suite *SystemMetricsSuite) SetupSuite() {
	suite.SuiteInitializer.Setup(suite.T())

	suite.metricsThingURL = util.GetThingURL(suite.Cfg.DigitalTwinAPIAddress, suite.ThingCfg.DeviceID)
	suite.metricsFeatureURL = util.GetFeatureURL(suite.metricsThingURL, metricsFeatureID)

	suite.topicData = util.GetLiveMessageTopic(suite.ThingCfg.DeviceID, protocol.TopicAction("data"))
}

func (suite *SystemMetricsSuite) TearDownSuite() {
	suite.SuiteInitializer.TearDown()
}

func (suite *SystemMetricsSuite) doTestMetrics(params map[string]interface{}) error {
	ws, err := util.NewDigitalTwinWSConnection(suite.Cfg)
	require.NoError(suite.T(), err, "failed to create websocket connection")
	defer ws.Close()

	err = util.SubscribeForWSMessages(suite.Cfg, ws, "START-SEND-MESSAGES", "")
	require.NoError(suite.T(), err, "unable to listen for events by using a websocket connection")

	defer util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", map[string]interface{}{
		"frequency": "0s",
	})

	_, err = util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", params)
	require.NoError(suite.T(), err, "error while executing operation")

	result := util.ProcessWSMessages(suite.Cfg, ws, func(msg *protocol.Envelope) (bool, error) {
		if strings.Index(msg.Path, "/outbox/") == -1 {
			return false, nil
		}

		if msg.Topic.String() != suite.topicData {
			return false, nil
		}

		data, err := json.Marshal(msg.Value)
		if err != nil {
			return false, nil
		}

		metric := new(metrics.MetricData)
		if err := json.Unmarshal(data, metric); err != nil {
			return false, nil
		}

		return true, nil
	})

	return result
}

func (suite *SystemMetricsSuite) TestMetricsRequest() {
	err := suite.doTestMetrics(map[string]interface{}{
		"frequency": "5s",
	})
	assert.NoError(suite.T(), err, "metrics event should be received")

	err = suite.doTestMetrics(map[string]interface{}{
		"frequency": "5s",
		"filter": []interface{}{map[string]interface{}{
			"originator": "SYSTEM",
		},
			map[string]interface{}{
				"id":         []string{"io.*", "cpu.*", "memory.*"},
				"originator": "suite-connector",
			},
		},
	})
	assert.NoError(suite.T(), err, "metrics event should be received")

	err = suite.doTestMetrics(map[string]interface{}{
		"frequency": "5s",
		"filter": []interface{}{map[string]interface{}{
			"id":         []string{"cpu.load", "cpu.load1", "cpu.load5", "cpu.load15"},
			"originator": "SYSTEM",
		},
			map[string]interface{}{
				"id":         []string{"io.*", "cpu.*", "memory.*"},
				"originator": "suite-connector",
			},
		},
	})
	assert.NoError(suite.T(), err, "metrics event should be received")
}

func (suite *SystemMetricsSuite) TestFilterNotMatching() {
	err := suite.doTestMetrics(map[string]interface{}{
		"frequency": "5s",
		"filter": []interface{}{map[string]interface{}{
			"id":         []string{"io.*", "cpu.*", "memory.*"},
			"originator": "test.process",
		},
		},
	})
	assert.Error(suite.T(), err, "metrics events should not be received")

	err = suite.doTestMetrics(map[string]interface{}{
		"frequency": "5s",
		"filter": []interface{}{map[string]interface{}{
			"id":         []string{"test.io", "test.cpu", "test.memory"},
			"originator": "SYSTEM",
		},
		},
	})
	assert.Error(suite.T(), err, "metrics events should not be received")
}

func (suite *SystemMetricsSuite) doTestMetricsError(params map[string]interface{}) {
	defer util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", map[string]interface{}{
		"frequency": "0s",
	})

	_, err := util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", params)
	assert.Error(suite.T(), err, "no error while executing operation")
}

func (suite *SystemMetricsSuite) TestMetricRequestWithUnknownUnit() {
	suite.doTestMetricsError(map[string]interface{}{
		"frequency": "10 seconds",
	})

	suite.doTestMetricsError(map[string]interface{}{
		"frequency": "100 seconds",
	})

	suite.doTestMetricsError(map[string]interface{}{
		"frequency": "2 hours",
	})
}

func TestSystemMetricsSuite(t *testing.T) {
	suite.Run(t, new(SystemMetricsSuite))
}
