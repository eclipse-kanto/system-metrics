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

package client

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/eclipse-kanto/system-metrics/internal/logger"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

const (
	topic = "edge/thing/response"
)

// BrokerConfig contains address and credentials for the MQTT broker
type BrokerConfig struct {
	Broker   string `json:"broker"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// EdgeConfiguration represents local Edge Thing configuration - its device, tenant and policy identifiers.
type EdgeConfiguration struct {
	DeviceID string `json:"deviceId"`
	TenantID string `json:"tenantId"`
	PolicyID string `json:"policyId"`
}

// EdgeManager represents the EdgeConnector used for connection with its configurations.
type EdgeManager struct {
	mqttClient    MQTT.Client
	edgeConfig    *EdgeConfiguration
	edgeConnector EdgeConnector
}

// EdgeConnector declares update and close function for every edge client.
type EdgeConnector interface {
	// Update updates the MQTT client and/or the configurations of the connector and connects it.
	Update(mqttClient MQTT.Client, edgeConfig *EdgeConfiguration)
	// Close disconnects the connector and the MQTT client and cleans up used resources.
	Close()
}

// NewEdgeManager creates EdgeManager with the given BrokerConfig
// and sets up a listener for EdgeConfiguration changes.
func NewEdgeManager(config *BrokerConfig, edgeConnector EdgeConnector) (*EdgeManager, error) {
	logger.Infof("Retrieve edge configuration from '%s'", config.Broker)

	opts := MQTT.NewClientOptions().
		AddBroker(config.Broker).
		SetClientID(uuid.New().String()).
		SetKeepAlive(30 * time.Second).
		SetCleanSession(true).
		SetAutoReconnect(true)
	if len(config.Username) > 0 {
		opts = opts.SetUsername(config.Username).SetPassword(config.Password)
	}

	edgeManager := &EdgeManager{mqttClient: MQTT.NewClient(opts)}

	if err := edgeManager.initEdgeConfigurationHandler(edgeConnector); err != nil {
		return nil, err
	}

	return edgeManager, nil
}

// initEdgeConfigurationHandler connects the MQTT Client and starts listening for Thing Info changes
func (edgeManager *EdgeManager) initEdgeConfigurationHandler(edgeConnector EdgeConnector) error {
	edgeManager.edgeConnector = edgeConnector

	if token := edgeManager.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	if token := edgeManager.mqttClient.Subscribe(topic, 1, func(client MQTT.Client, message MQTT.Message) {
		localCfg := &EdgeConfiguration{}
		err := json.Unmarshal(message.Payload(), localCfg)
		if err != nil {
			logger.Error("Could not unmarshal edge configuration", err)
			return
		}

		if edgeManager.edgeConfig == nil || *edgeManager.edgeConfig != *localCfg {
			logger.Infof("New edge configuration received")
			if edgeManager.edgeConfig != nil {
				edgeManager.edgeConnector.Close()
			}

			edgeManager.edgeConfig = localCfg
			edgeManager.edgeConnector.Update(edgeManager.mqttClient, edgeManager.edgeConfig)
		}
	}); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	if token := edgeManager.mqttClient.Publish("edge/thing/request", 1, false, ""); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

// Validate validates the Broker config.
func (b *BrokerConfig) Validate() error {
	if len(b.Broker) == 0 {
		return errors.New("broker is missing")
	}

	return nil
}

// Close closes the EdgeConnector
func (edgeManager *EdgeManager) Close() {
	if edgeManager.edgeConfig != nil {
		edgeManager.edgeConnector.Close()
	}

	edgeManager.mqttClient.Unsubscribe(topic)
	edgeManager.mqttClient.Disconnect(200)

	logger.Info("Disconnected from MQTT broker")
}
