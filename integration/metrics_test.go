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
	"fmt"
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
	dataAction       = "data"
)

type systemMetricsSuite struct {
	suite.Suite

	util.SuiteInitializer

	metricsThingURL   string
	metricsFeatureURL string

	pathData  string
	topicData string
}

func (suite *systemMetricsSuite) SetupSuite() {
	suite.SuiteInitializer.Setup(suite.T())

	suite.metricsThingURL = util.GetThingURL(suite.Cfg.DigitalTwinAPIAddress, suite.ThingCfg.DeviceID)
	suite.metricsFeatureURL = util.GetFeatureURL(suite.metricsThingURL, metricsFeatureID)

	suite.pathData = util.GetFeatureOutboxMessagePath(metricsFeatureID, dataAction)
	suite.topicData = util.GetLiveMessageTopic(suite.ThingCfg.DeviceID, protocol.TopicAction(dataAction))
}

func (suite *systemMetricsSuite) TearDownSuite() {
	suite.SuiteInitializer.TearDown()
}

func TestSystemMetricsSuite(t *testing.T) {
	suite.Run(t, new(systemMetricsSuite))
}

func (suite *systemMetricsSuite) testMetrics(params map[string]interface{}, originators ...string) error {
	ws, err := util.NewDigitalTwinWSConnection(suite.Cfg)
	require.NoError(suite.T(), err, "failed to create websocket connection")
	defer ws.Close()

	err = util.SubscribeForWSMessages(suite.Cfg, ws, "START-SEND-MESSAGES", "")
	require.NoError(suite.T(), err, "unable to listen for events by using a websocket connection")

	defer func() {
		_, err := util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", map[string]interface{}{
			"frequency": "0s",
		})
		assert.NoError(suite.T(), err, "error while stopping system metrics")
	}()

	_, err = util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", params)
	require.NoError(suite.T(), err, "error while requesting the system metrics")

	result := util.ProcessWSMessages(suite.Cfg, ws, func(msg *protocol.Envelope) (bool, error) {
		if msg.Path != suite.pathData {
			return false, nil
		}

		if msg.Topic.String() != suite.topicData {
			return false, nil
		}

		data, err := json.Marshal(msg.Value)
		if err != nil {
			return true, err
		}

		metric := new(metrics.MetricData)
		if err := json.Unmarshal(data, metric); err != nil {
			return true, err
		}

		var originatorFound bool
		for _, m := range metric.Snapshot {

			for _, originator := range originators {
				if originator == m.Originator {
					originatorFound = true
				}
			}

			for _, mm := range m.Measurements {
				if strings.HasPrefix(mm.ID, "cpu.") {
					continue
				}

				if strings.HasPrefix(mm.ID, "memory.") {
					continue
				}

				if strings.HasPrefix(mm.ID, "io.") {
					continue
				}

				if strings.HasPrefix(mm.ID, "test.") {
					continue
				}

				return false, fmt.Errorf("Invalid metrics ID: %s", mm.ID)
			}
		}

		return originatorFound, nil
	})

	return result
}

func (suite *systemMetricsSuite) TestMetricsRequestDefaultOriginator() {
	err := suite.testMetrics(map[string]interface{}{
		"frequency": "5s",
	}, "SYSTEM")
	assert.NoError(suite.T(), err, "metrics event from system originator should be received")
}

func (suite *systemMetricsSuite) TestMetricsRequestMultipleOriginators() {
	err := suite.testMetrics(map[string]interface{}{
		"frequency": "5s",
		"filter": []interface{}{map[string]interface{}{
			"originator": "SYSTEM",
		},
			map[string]interface{}{
				"id":         []string{"io.*", "cpu.*", "memory.*"},
				"originator": "suite-connector",
			},
		},
	}, "SYSTEM", "suite-connector")
	assert.NoError(suite.T(), err, "metrics event from at least one originator system/suite-connector should be received")
}

func (suite *systemMetricsSuite) TestMetricsRequestSystemLoadAverage() {
	err := suite.testMetrics(map[string]interface{}{
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
	}, "SYSTEM", "suite-connector")
	assert.NoError(suite.T(), err, "metrics event should be received")
}

func (suite *systemMetricsSuite) TestFilterNotMatching() {
	err := suite.testMetrics(map[string]interface{}{
		"frequency": "5s",
		"filter": []interface{}{map[string]interface{}{
			"id":         []string{"io.*", "cpu.*", "memory.*"},
			"originator": "test.process",
		},
		},
	}, "test.process")
	assert.Error(suite.T(), err, "metrics events for test.process should not be received")

	err = suite.testMetrics(map[string]interface{}{
		"frequency": "5s",
		"filter": []interface{}{map[string]interface{}{
			"id":         []string{"test.io", "test.cpu", "test.memory"},
			"originator": "SYSTEM",
		},
		},
	}, "SYSTEM")
	assert.Error(suite.T(), err, "metrics events for non existing measurements test.* should not be received")
}

func (suite *systemMetricsSuite) testMetricsError(params map[string]interface{}) {
	defer func() {
		_, err := util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", map[string]interface{}{
			"frequency": "0s",
		})
		assert.NoError(suite.T(), err, "error while stopping system metrics")
	}()

	_, err := util.ExecuteOperation(suite.Cfg, suite.metricsFeatureURL, "request", params)
	assert.Error(suite.T(), err, "no error while requesting the system metrics")
}

func (suite *systemMetricsSuite) TestMetricRequestWithInvalidUnit() {
	suite.testMetricsError(map[string]interface{}{
		"frequency": "10 seconds", // Valid is 10s
	})

	suite.testMetricsError(map[string]interface{}{
		"frequency": "2 hours", // Valid is 2h
	})
}
