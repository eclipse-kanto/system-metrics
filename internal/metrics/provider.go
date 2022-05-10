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
	"fmt"
	"sync"

	"time"

	"github.com/eclipse-kanto/system-metrics/internal/logger"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type metricsConsumer = func(m MetricData, correlationID string)

type provider struct {
	request *Request
	metrics *MetricData

	consumer metricsConsumer

	cronData *cron.Cron
	cronID   cron.EntryID

	mutex sync.Mutex
}

func newProvider(mc metricsConsumer, initialRequest *Request) *provider {
	if initialRequest != nil {
		initialRequest.normalize()
	}

	return &provider{
		consumer: mc,
		request:  initialRequest,
	}
}

// MetricData contains a snapshot with all originators' measurements collected at a concrete time.
type MetricData struct {
	Snapshot  []OriginatorMeasurements `json:"snapshot"`
	Timestamp int64                    `json:"timestamp"`
}

// OriginatorMeasurements represents all the measurements collected per originator.
type OriginatorMeasurements struct {
	Originator   string        `json:"originator"`
	Measurements []Measurement `json:"measurements"`
}

// Measurement represents a measured value per metric ID.
type Measurement struct {
	ID    string  `json:"id"`
	Value float64 `json:"value"`
}

func (p *provider) handleRequest(mRequest *Request) error {
	// remove current process if any
	p.cancelProcess()

	// start the new request process
	p.request = mRequest
	p.metrics = &MetricData{}
	if err := p.startProcess(); err != nil {
		return err
	}

	return nil
}

func (p *provider) startProcess() error {
	if p.cronData == nil {
		p.cronData = cron.New(cron.WithSeconds())
		p.cronData.Start()
	}

	if p.request != nil && p.request.Frequency.Duration > 0 {
		entryID, err := p.cronData.AddFunc(fmt.Sprintf("@every %s", p.request.Frequency.Duration), p.loadAndSendMetrics)
		p.cronID = entryID
		if err != nil {
			return errors.Wrapf(err, "unexpected frequency '%s'", p.request.Frequency.Duration)
		}
		logger.Infof("Request handling started with frequency '%s', scheduler ID %v, filter %v",
			p.request.Frequency.Duration, p.cronID, p.request.Filter)

	} else {
		logger.Debug("No active request")
	}

	return nil
}

func (p *provider) cancelProcess() {
	if p.cronData != nil && p.cronID != 0 {
		p.cronData.Remove(p.cronID)
		logger.Infof("Scheduler with ID %v is removed", p.cronID)
		p.cronID = 0
	}
}

// load metrics for the available filters and send them with response message
func (p *provider) loadAndSendMetrics() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	logger.Trace("Start new metrics data snapshot")

	p.metrics = &MetricData{
		Snapshot:  make([]OriginatorMeasurements, 0),
		Timestamp: time.Now().Unix(),
	}

	// system metrics
	if p.request.hasFilterFor(originatorSystem) {
		systemData := &OriginatorMeasurements{
			Originator:   originatorSystem,
			Measurements: make([]Measurement, 0),
		}

		p.addCPUMetrics(systemData)
		p.addMemMetrics(systemData)

		p.appendOriginatorMeasurements(systemData)
	}

	// filtered processes metrics
	p.addProcessesMetrics()

	// log and send if there is collected data
	if len(p.metrics.Snapshot) > 0 {
		if logger.IsTraceEnabled() {
			traceAsJSON("Metrics data to send: %s", p.metrics)
		}

		//send live message to local broker
		if p.consumer != nil {
			p.consumer(*p.metrics, p.request.correlationID)
		}
	} else {
		logger.Trace("No metrics data collected")
	}
}

func (p *provider) addCPUMetrics(systemData *OriginatorMeasurements) {
	// utilization
	if p.request.hasFilterForItem(cpuUtilization, originatorSystem) {
		if utilization, err := cpu.Percent(0, false); err != nil {
			logger.Errorf("Failed to get CPU Utilization metrics: %v", err)
		} else {
			appendMeasurement(systemData, cpuUtilization, utilization[0])
		}
	}

	// load
	if p.request.hasFilterForGroups(originatorSystem, cpuLoadGroupPrefix, "cpu.*") {
		if load, err := load.Avg(); err != nil {
			logger.Errorf("Failed to get CPU Load metrics: %v", err)
		} else {
			p.appendMeasurementOpt(systemData, cpuLoad1, load.Load1)
			p.appendMeasurementOpt(systemData, cpuLoad5, load.Load5)
			p.appendMeasurementOpt(systemData, cpuLoad15, load.Load15)
		}
	}
}

func (p *provider) addMemMetrics(systemData *OriginatorMeasurements) {
	if p.request.hasFilterForGroups(originatorSystem, memoryGroupPrefix) {
		v, err := mem.VirtualMemory()
		if err != nil {
			logger.Error("Unable to access Virtual Memory data", err)
			return
		}

		p.appendMeasurementOpt(systemData, memoryTotal, float64(v.Total))
		p.appendMeasurementOpt(systemData, memoryAvailable, float64(v.Available))
		p.appendMeasurementOpt(systemData, memoryUsed, float64(v.Used))
		p.appendMeasurementOpt(systemData, memoryUtilization, v.UsedPercent)
	}
}

func (p *provider) addProcessesMetrics() {
	for _, f := range p.request.Filter {
		if len(f.Originator) != 0 && f.Originator != originatorSystem {
			pid, err := processPIDOf(f.Originator)
			if err != nil {
				logger.Warningf("No process metrics for %s", f.Originator)
				continue
			}

			if process, err := process.NewProcess(pid); err != nil {
				logger.Warningf("Unable to access procees data for name %s: %v", f.Originator, err)
			} else {
				p.addProcessMetrics(process, f.Originator)
			}
		}
	}
}

func (p *provider) addProcessMetrics(process *process.Process, originator string) {
	originatorData := &OriginatorMeasurements{
		Originator:   originator,
		Measurements: make([]Measurement, 0),
	}
	if p.request.hasFilterForItem(cpuUtilization, originator) {
		cpu, err := process.CPUPercent()
		if err != nil {
			logger.Errorf("Failed to get CPU Utilization for process %s: %v", originator, err)
		} else {
			appendMeasurement(originatorData, cpuUtilization, cpu)
		}
	}

	if p.request.hasFilterForItem(memoryUtilization, originator) {
		memory, err := process.MemoryPercent()
		if err != nil {
			logger.Errorf("Failed to get Memory Utilization for process %s: %v", originator, err)
		} else {
			appendMeasurement(originatorData, memoryUtilization, float64(memory))
		}
	}

	if p.request.hasFilterForGroups(originator, ioGroupPrefix) {
		bytes, err := process.IOCounters()
		if err != nil {
			logger.Errorf("Failed to get IO bytes for process %s: %v", originator, err)
		} else {
			p.appendMeasurementOpt(originatorData, ioReadBytes, float64(bytes.ReadBytes))
			p.appendMeasurementOpt(originatorData, ioWriteBytes, float64(bytes.WriteBytes))
		}
	}

	p.appendOriginatorMeasurements(originatorData)
}

func (p *provider) appendOriginatorMeasurements(om *OriginatorMeasurements) {
	if len(om.Measurements) > 0 {
		p.metrics.Snapshot = append(p.metrics.Snapshot, *om)
	}
}

func appendMeasurement(om *OriginatorMeasurements, ID string, value float64) {
	om.Measurements = append(om.Measurements, Measurement{ID, value})
}

func (p *provider) appendMeasurementOpt(om *OriginatorMeasurements, ID string, value float64) {
	if p.request.hasFilterForItem(ID, om.Originator) {
		appendMeasurement(om, ID, value)
	}
}

// processPIDOf returns the PID of the process with the provided name.
func processPIDOf(name string) (int32, error) {
	processes, _ := process.Processes()
	for _, ps := range processes {
		psName, _ := ps.Name()
		if psName == name {
			return ps.Pid, nil
		}
	}
	return -1, errors.New("process not found")
}
