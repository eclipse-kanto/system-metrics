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

package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/eclipse-kanto/system-metrics/internal/client"
	"github.com/eclipse-kanto/system-metrics/internal/logger"
	"github.com/eclipse/ditto-clients-golang"
	"github.com/eclipse/ditto-clients-golang/model"
	"github.com/eclipse/ditto-clients-golang/protocol"
	"github.com/eclipse/ditto-clients-golang/protocol/things"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	featureID              = "Metrics"
	featureDefinition      = "com.bosch.iot.suite.edge.metric:Metrics:1.0.0"
	subjectMetricsResponse = "data"
	pathMetricsRequest     = "/features/" + featureID + "/inbox/messages/request"
	contentTypeJSON        = "application/json"

	defaultDisconnectTimeout = 250 * time.Millisecond
	defaultKeepAlive         = 20 * time.Second
)

// Metrics feature implementation.
type Metrics struct {
	client *ditto.Client

	deviceID string
	tenantID string

	initialRequest *Request

	provider *provider

	mutex sync.Mutex
}

// NewMetrics constructs Metrics with an initial request.
func NewMetrics(initialRequest *Request) *Metrics {
	m := &Metrics{initialRequest: initialRequest}
	m.provider = newProvider(m.sendResponse, m.initialRequest)

	return m
}

// Update Metrics configuration and connect to the Ditto endpoint.
func (m *Metrics) Update(mqttClient MQTT.Client, edgeConfig *client.EdgeConfiguration) {
	m.initNewEdgeConfiguration(edgeConfig)

	config := ditto.NewConfiguration().
		WithDisconnectTimeout(defaultDisconnectTimeout).
		WithConnectHandler(
			func(client *ditto.Client) {
				m.connectHandler(client)
			})

	var err error
	m.client, err = ditto.NewClientMQTT(mqttClient, config)
	if err != nil {
		logger.Errorf("Couldn't create MQTT client: %v", err)
		return
	}

	m.client.Subscribe(m.messageHandlerFunc())

	if err = m.client.Connect(); err != nil {
		logger.Errorf("Couldn't connect MQTT client: %v", err)
		return
	}

	logger.Tracef("Connect to device '%s'", m.deviceID)
}

func (m *Metrics) initNewEdgeConfiguration(edgeConfig *client.EdgeConfiguration) {
	m.provider.request = m.initialRequest

	m.deviceID = edgeConfig.DeviceID
	m.tenantID = edgeConfig.TenantID
}

func (m *Metrics) messageHandlerFunc() ditto.Handler {
	return func(requestID string, msg *protocol.Envelope) {
		go m.messageHandler(requestID, msg)
	}
}

// Close Metrics by stopping the provider and disconnecting from the Ditto endpoint and cleaning up used resources.
func (m *Metrics) Close() {
	logger.Tracef("Disconnect to device '%s'", m.deviceID)
	m.client.Unsubscribe()
	m.stopProvider()

	m.client.Disconnect()
}

// Create feature on connect

func (m *Metrics) connectHandler(client *ditto.Client) {
	cmd := things.NewCommand(model.NewNamespacedIDFrom(m.deviceID)).
		Feature(featureID).
		Modify((&model.Feature{}).WithDefinitionFrom(featureDefinition))
	msg := cmd.Envelope(
		protocol.WithResponseRequired(false),
		protocol.WithCorrelationID(uuid.NewString()))

	err := client.Send(msg)
	if err != nil {
		panic(fmt.Errorf("failed to send '%s' feature modify command with correlation id '%s'",
			featureID, msg.Headers.CorrelationID()))
	}
	logger.Infof("'%s' feature modify command is sent successfully", featureID)

	// start metrics upload for the last request if any
	m.startProvider()
}

// Handle metrics request

func (m *Metrics) messageHandler(requestID string, msg *protocol.Envelope) {
	if model.NewNamespacedID(msg.Topic.Namespace, msg.Topic.EntityName).String() == m.deviceID {
		if msg.Path == pathMetricsRequest {
			correlationID := msg.Headers.CorrelationID()
			msgErr := m.handleMessageRequest(msg, correlationID)
			if len(correlationID) > 0 {
				m.sendReply(requestID, correlationID, string(msg.Topic.Action), msgErr)
			}
		} else {
			logger.Debugf("Unknown message received with path %s", msg.Path)
		}
	}
}

func (m *Metrics) handleMessageRequest(msg *protocol.Envelope, correlationID string) *MessageError {
	metricsRequest, err := fetchRequest(msg.Value)
	if err != nil {
		logger.Errorf("Unexpected metrics request message with correlationId %s, err: %v", correlationID, err)
		return NewMessageParameterInvalidError()
	}

	logger.Debugf("Received metrics request %v", metricsRequest.Filter)
	metricsRequest.correlationID = correlationID
	err = m.provider.handleRequest(metricsRequest)
	if err != nil {
		logger.Errorf(
			"Unable to proceed metrics request message with correlationId %s, err: %v", correlationID, err)
		return NewMessageInternalError()
	}
	return nil
}

func fetchRequest(data interface{}) (*Request, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid payload: %v", data)
	}

	request := &Request{}
	if err = json.Unmarshal(payload, request); err != nil {
		return nil, errors.Wrapf(err, "invalid content type: %v", data)
	}

	request.normalize()
	return request, nil
}

// Send request response

func (m *Metrics) sendReply(requestID, correlationID, action string, msgErr *MessageError) {
	env := m.replyEnvelope(correlationID, action, msgErr)
	if err := m.client.Reply(requestID, env); err != nil {
		logger.Tracef("Failed to send error response to request Id %s: %v", requestID, err)
	} else {
		traceAsJSON("Sent reply %s", *env)
	}
}

func (m *Metrics) replyEnvelope(correlationID, action string, msgErr *MessageError) *protocol.Envelope {
	statusMsg := things.NewMessage(model.NewNamespacedIDFrom(m.deviceID)).
		Feature(featureID).
		Outbox(action)

	headersOpts := []protocol.HeaderOpt{
		protocol.WithCorrelationID(correlationID),
		protocol.WithResponseRequired(false)}
	if msgErr == nil {
		return statusMsg.Envelope(headersOpts...).WithStatus(http.StatusNoContent)
	}

	headersOpts = append(headersOpts, protocol.WithContentType(contentTypeJSON))
	return statusMsg.Envelope(headersOpts...).WithValue(msgErr).WithStatus(msgErr.Status)
}

// Send metrics responses

func (m *Metrics) sendResponse(data MetricData, correralitionID string) {
	msg := things.NewMessage(model.NewNamespacedIDFrom(m.deviceID)).
		Feature(featureID).
		Outbox(subjectMetricsResponse).
		WithPayload(data)

	if m.client == nil {
		res, _ := json.Marshal(data)
		logger.Errorf("No active client to send data %s", string(res))
		return
	}

	headerOpts := []protocol.HeaderOpt{
		protocol.WithResponseRequired(false),
		protocol.WithContentType(contentTypeJSON),
	}
	if len(correralitionID) > 0 {
		headerOpts = append(headerOpts, protocol.WithCorrelationID(correralitionID))
	}

	err := m.client.Send(msg.Envelope(headerOpts...))
	if err != nil {
		logger.Error("Failed to send message", err)
	} else {
		logger.Debugf("Message is send %v", *msg)
	}
}

func (m *Metrics) startProvider() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.provider.startProcess()
}

func (m *Metrics) stopProvider() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.provider != nil {
		m.provider.cancelProcess()
	}
}

func traceAsJSON(format string, a interface{}) {
	if logger.IsTraceEnabled() {
		res, _ := json.Marshal(a)
		logger.Tracef(format, string(res))
	}
}
