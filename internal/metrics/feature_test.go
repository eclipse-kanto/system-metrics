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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/eclipse-kanto/system-metrics/internal/client"
	"github.com/eclipse-kanto/system-metrics/internal/logger"
	"github.com/eclipse/ditto-clients-golang"
	"github.com/eclipse/ditto-clients-golang/model"
	"github.com/eclipse/ditto-clients-golang/protocol"
	"github.com/eclipse/ditto-clients-golang/protocol/things"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	testThingID          = "test:thingID"
	testTenantID         = "testTenantID"
	testFeatureID        = "Metrics"
	testOperationRequest = "request"
	inboxPathPrefix      = "/features/" + testFeatureID + "/inbox"
	outboxPathPrefix     = "/features/" + testFeatureID + "/outbox"
)

func TestFetchMetricsRequest(t *testing.T) {
	tests := []struct {
		name    string
		arg     interface{}
		want    *Request
		wantErr bool
	}{
		{
			name:    "testInvalidAsString",
			arg:     "invalid",
			wantErr: true,
		},
		{
			name: "testMapUnknownKeys",
			arg:  map[string]int{"foo": 1},
			want: &Request{
				Frequency: Duration{0 * time.Second},
			},
			wantErr: false,
		},
		{
			name: "testFrequencyOnly",
			arg:  map[string]interface{}{"frequency": "2s"},
			want: &Request{
				Frequency: Duration{2 * time.Second},
				Filter: []Filter{{
					ID:         []string{cpuUtilization, memoryUtilization},
					Originator: originatorSystem,
				}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchRequest(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchMetricsRequest error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchMetricsRequest got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleMetricsRequest(t *testing.T) {
	tests := []struct {
		name    string
		arg     *Request
		process string
		wait    time.Duration
		expErr  bool
	}{
		{
			name: "testSystemMetricsRequest",
			arg: &Request{
				Frequency: Duration{1 * time.Second},
				Filter: []Filter{{
					ID:         []string{cpuUtilization, memoryUtilization},
					Originator: originatorSystem,
				}},
			},
			process: originatorSystem,
			wait:    3 * time.Second,
			expErr:  false,
		},
		{
			name: "testSystemMetricsRequestAll",
			arg: &Request{
				Frequency: Duration{1 * time.Second},
				Filter: []Filter{{
					ID:         []string{"cpu.*", "memory.*"},
					Originator: originatorSystem,
				}},
			},
			process: originatorSystem,
			wait:    3 * time.Second,
			expErr:  false,
		},
		{
			name: "testProcessMetricsRequest",
			arg: &Request{
				Frequency: Duration{1 * time.Second},
				Filter: []Filter{{
					ID:         []string{"io.*", "cpu.*", "memory.*"},
					Originator: testingProcess(),
				}},
			},
			process: testingProcess(),
			wait:    3 * time.Second,
			expErr:  false,
		},
	}
	tm, mc := testInit()
	doTestConnection(tm, mc)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.process) == 0 {
				t.Skipf("no permission to run '%s'", tt.name)
			}

			err := tm.provider.handleRequest(tt.arg)
			if (err != nil) != tt.expErr {
				t.Errorf("handleRequest for = %v, error = %v, wantErr %v", tt.arg, err, tt.expErr)
			}
			time.Sleep(tt.wait)
			tm.stopProvider()
		})
	}
}

func TestReplyInvalidParameterError(t *testing.T) {
	paramInvalid := NewMessageParameterInvalidError()
	correlationID := "CorrelationID_InvalidParameterError"

	metrics, mc := testInit()
	doTestConnection(metrics, mc)
	env := metrics.replyEnvelope(correlationID, testOperationRequest, NewMessageParameterInvalidError())
	assertResponseHeaders(t, "TestRequestInvalidParameterError", correlationID, true, env.Headers.Values)
	if !reflect.DeepEqual(paramInvalid, env.Value) {
		t.Errorf("Reply value does not match: %v != %v", paramInvalid, env.Value)
	}
	if env.Status != 400 {
		t.Errorf("Reply status does not match: 400 != %v", env.Status)
	}
}

func TestReplyOK(t *testing.T) {
	correlationID := "CorrelationID_Valid"

	metrics, mc := testInit()
	doTestConnection(metrics, mc)
	env := metrics.replyEnvelope(correlationID, testOperationRequest, nil)
	assertResponseHeaders(t, "TestRequestValid", correlationID, false, env.Headers.Values)
	if env.Status != 204 {
		t.Errorf("Reply status does not match: 204 != %v", env.Status)
	}
}

func TestHandleRequestPublishedMessages(t *testing.T) {
	metrics, mc := testInit()
	doTestConnection(metrics, mc)
	assertNoMetricData(t, mc)

	msg := things.NewMessage(model.NewNamespacedIDFrom(testThingID)).
		Feature(featureID).
		WithPayload(&Request{
			Frequency: Duration{1 * time.Second},
		})

	metrics.client.Send(msg.Inbox(testOperationRequest).
		Envelope(protocol.WithResponseRequired(true),
			protocol.WithCorrelationID("request_1")))

	assertReplyStatusOK(t, mc)
	assertDefaultMetricData(t, mc)

	msgStop := things.NewMessage(model.NewNamespacedIDFrom(testThingID)).
		Feature(featureID).
		WithPayload(&Request{
			Frequency: Duration{0 * time.Second},
		})

	metrics.client.Send(msgStop.Inbox(testOperationRequest).
		Envelope(protocol.WithResponseRequired(true),
			protocol.WithCorrelationID("request_stop")))

	assertReplyStatusOK(t, mc)
	assertNoMetricData(t, mc)

	metricsClose(t, metrics, mc)
}

func TestHandleRequestInvalidFrequency(t *testing.T) {
	metrics, mc := testInit()
	doTestConnection(metrics, mc)

	msg := things.NewMessage(model.NewNamespacedIDFrom(testThingID)).
		Feature(featureID).
		WithPayload("{'frequency':'invalid_duration'}")

	metrics.client.Send(msg.Inbox(testOperationRequest).
		Envelope(protocol.WithResponseRequired(true),
			protocol.WithCorrelationID("request_invalid_duration")))

	receivedMsg := mc.value(t)
	format := "expected message error %v but is: %v"
	if receivedMsg == nil {
		t.Fatalf(format, NewMessageParameterInvalidError(), receivedMsg)
	} else {
		if len(receivedMsg.Value.(map[string]interface{})) == 0 {
			t.Fatalf(format, NewMessageParameterInvalidError(), receivedMsg.Value)
		}
	}

	metricsClose(t, metrics, mc)
}

func TestHandleRequestUnknownPath(t *testing.T) {
	metrics, mc := testInit()
	doTestConnection(metrics, mc)

	msg := things.NewMessage(model.NewNamespacedIDFrom(testThingID)).
		Feature(featureID)
	metrics.client.Send(msg.Inbox("unknownPath").Envelope(protocol.WithCorrelationID("request_unknown_path")))
	assertNoMetricData(t, mc)

	metricsClose(t, metrics, mc)
}

func testInit() (*Metrics, *mockedPahoClient) {
	logger.SetupLogger(&logger.LogConfig{LogLevel: "TRACE"})

	mc := &mockedPahoClient{payload: make(chan interface{}, 1)}
	sm := NewMetrics(nil)

	mc = mc.WithMessageHandler(sm.messageHandlerFunc())
	return sm, mc
}

func doTestConnection(tm *Metrics, mc *mockedPahoClient) {
	edgeCfg := &client.EdgeConfiguration{DeviceID: testThingID, TenantID: testTenantID}
	tm.Update(mc, edgeCfg)
}

// Assertion utilities

func assertResponseHeaders(
	t *testing.T, test string, correlationID string, withContent bool, values map[string]interface{},
) {
	headersLen := 2
	if withContent {
		headersLen = 3
	}
	if len(values) != headersLen {
		t.Errorf("%s reply headers length does not match: %v != %v", test, headersLen, len(values))
	}
	if values[protocol.HeaderCorrelationID] != correlationID {
		t.Errorf("%s reply header 'correlation-id' does not match: %v != %v", test, correlationID, values)
	}
	if values[protocol.HeaderResponseRequired] != false {
		t.Errorf("%s reply header 'response-required' does not match: false != %v", test, values)
	}
	if withContent && values[protocol.HeaderContentType] != "application/json" {
		t.Errorf("%s reply header 'content-type' does not match: application/json != %v", test, values)
	}
}

func metricsClose(t *testing.T, metrics *Metrics, mc *mockedPahoClient) {
	metrics.Close()
	assertNoMetricData(t, mc)
}

func assertNoMetricData(t *testing.T, mc *mockedPahoClient) {
	receivedMsg := mc.value(t)
	if receivedMsg != nil {
		t.Fatalf("expected no metrics: %v", receivedMsg)
	}
}

// assertDefaultMetricData validates if system CPU and memory utilizations are received
func assertDefaultMetricData(t *testing.T, mc *mockedPahoClient) {
	receivedMsg := mc.value(t)
	if receivedMsg == nil {
		t.Fatalf("expected message with metrics and no error: %v", receivedMsg)
	}
	value := receivedMsg.Value
	if value == nil {
		t.Fatalf("expected message with metrics and no error: %v", value)
	}

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("expected payload: %v", value)
	}
	var data MetricData
	if err = json.Unmarshal(payload, &data); err != nil {
		t.Fatalf("expected metrics as payload: %v", value)
	}

	if data.Timestamp <= 0 {
		t.Fatalf("invalid metrics data timetamp: %v", data.Timestamp)
	}
	if len(data.Snapshot) != 1 {
		t.Fatalf("'%s' measurements expected, received: %v", originatorSystem, data.Snapshot)
	}
	if data.Snapshot[0].Originator != originatorSystem {
		t.Fatalf("originator '%s' expected,  received: %v", originatorSystem, data.Snapshot[0].Originator)
	}
	if len(data.Snapshot[0].Measurements) != 2 {
		t.Fatalf("unexpected system originator metrics received: %v", data.Snapshot)
	}
}

func assertReplyStatusOK(t *testing.T, mc *mockedPahoClient) {
	receivedMsg := mc.value(t)
	if receivedMsg == nil {
		t.Fatalf("expected sent message to be with status 204 and no error: %v", receivedMsg)
	}
	value := receivedMsg.Value
	if value != nil {
		t.Fatalf("expected sent message to be with status 204 and no error: %v", receivedMsg)
	}
}

// Testing MQTT Client

// mockedPahoClient represents mocked mqtt.Token interface used for testing.
type mockedPahoClient struct {
	err            error
	payload        chan interface{}
	messageHandler ditto.Handler
}

func (client *mockedPahoClient) WithMessageHandler(handler ditto.Handler) *mockedPahoClient {
	client.messageHandler = handler
	return client
}

// value returns last payload value or waits 1 second for new payload.
func (client *mockedPahoClient) value(t *testing.T) *protocol.Envelope {
	select {
	case payload := <-client.payload:
		env := &protocol.Envelope{}
		if err := json.Unmarshal(payload.([]byte), env); err != nil {
			t.Fatalf("unexpected error during data unmarshal: %v", err)
		}
		if !strings.HasPrefix(env.Path, outboxPathPrefix) {
			t.Fatalf("message path do not starts with [%v]: %v", outboxPathPrefix, env.Path)
		}
		return env
	case <-time.After(1 * time.Second):
	}
	return nil
}

// IsConnected returns true.
func (client *mockedPahoClient) IsConnected() bool {
	return true
}

// IsConnectionOpen returns true.
func (client *mockedPahoClient) IsConnectionOpen() bool {
	return true
}

// Connect returns finished token.
func (client *mockedPahoClient) Connect() mqtt.Token {
	return &mockedToken{err: client.err}
}

// Disconnect do nothing.
func (client *mockedPahoClient) Disconnect(quiesce uint) {
	// Do nothing.
}

// Publish returns finished token and set client topic and payload.
func (client *mockedPahoClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	env := &protocol.Envelope{}
	if err := json.Unmarshal(payload.([]byte), env); err == nil {
		if strings.HasPrefix(env.Path, outboxPathPrefix) {
			client.payload <- payload
		}
		if client.messageHandler != nil && strings.HasPrefix(env.Path, inboxPathPrefix) {
			go client.messageHandler("", env)
		}
	}
	return &mockedToken{err: client.err}
}

// Subscribe returns finished token.
func (client *mockedPahoClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return &mockedToken{err: client.err}
}

// SubscribeMultiple returns finished token.
func (client *mockedPahoClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	return &mockedToken{err: client.err}
}

// Unsubscribe returns finished token.
func (client *mockedPahoClient) Unsubscribe(topics ...string) mqtt.Token {
	return &mockedToken{err: client.err}
}

// AddRoute do nothing.
func (client *mockedPahoClient) AddRoute(topic string, callback mqtt.MessageHandler) {
	// Do nothing.
}

// OptionsReader returns an empty struct.
func (client *mockedPahoClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{}
}

// mockedToken represents mocked mqtt.Token interface used for testing.
type mockedToken struct {
	err error
}

// Wait returns immediately with true.
func (token *mockedToken) Wait() bool {
	return true
}

// WaitTimeout returns immediately with true.
func (token *mockedToken) WaitTimeout(time.Duration) bool {
	return true
}

// Done returns immediately with nil channel.
func (token *mockedToken) Done() <-chan struct{} {
	return nil
}

// Error returns the error if set.
func (token *mockedToken) Error() error {
	return token.err
}
