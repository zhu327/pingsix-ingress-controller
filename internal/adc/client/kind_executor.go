// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/api7/etcd-adapter/pkg/adapter"
	"github.com/go-logr/logr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/kine"
)

const (
	// etcdAdapterAddr is the address for the local etcd adapter
	etcdAdapterAddr = "127.0.0.1:12379"
	// apisixKeyPrefix is the prefix for all APISIX resources in etcd
	apisixKeyPrefix = "/apisix"
)

// KindExecutor implements ADCExecutor interface using Kine to sync resources
type KindExecutor struct {
	log logr.Logger

	cache   kine.Cache
	differ  kine.Differ
	adapter adapter.Adapter
}

func newEtcdAdapter(log logr.Logger) adapter.Adapter {
	a := adapter.NewEtcdAdapter(nil)

	ln, err := net.Listen("tcp", etcdAdapterAddr)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := a.Serve(context.Background(), ln); err != nil {
			panic(err)
		}
		log.Info("etcd adapter started")
	}()

	return a
}

// NewKindExecutor creates a new KindExecutor
func NewKindExecutor(log logr.Logger) *KindExecutor {
	cache, err := kine.NewMemDBCache()
	if err != nil {
		panic(err)
	}
	differ := kine.NewDiffer(cache)
	return &KindExecutor{
		log:     log,
		cache:   cache,
		differ:  differ,
		adapter: newEtcdAdapter(log),
	}
}

func (e *KindExecutor) Execute(ctx context.Context, config adctypes.Config, args []string) error {
	return e.runKindSync(ctx, config, args)
}

func (e *KindExecutor) runKindSync(ctx context.Context, config adctypes.Config, args []string) error {
	// Parse args to extract labels, types, and file path
	labels, adcTypes, filePath, err := e.parseArgs(args)
	if err != nil {
		return fmt.Errorf("failed to parse args: %w", err)
	}

	// Load resources from file
	resources, err := e.loadResourcesFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load resources from file %s: %w", filePath, err)
	}

	// Transfer ADC resources to Kine resources
	e.log.V(1).Info("transferring ADC resources to Kine resources")
	transferredResources, err := kine.TransferResources(resources)
	if err != nil {
		return fmt.Errorf("failed to transfer resources: %w", err)
	}

	// Convert ADC types to Kine types
	kineTypes := e.convertADCTypesToKineTypes(adcTypes)

	// Generate diff events
	e.log.V(1).Info("generating diff events")
	diffOpts := &kine.DiffOptions{
		Labels: labels,
		Types:  kineTypes,
	}
	events, err := e.differ.Diff(transferredResources, diffOpts)
	if err != nil {
		return fmt.Errorf("failed to diff resources: %w", err)
	}

	e.log.Info("diff completed", "totalEvents", len(events))

	// Process events: apply cache changes and send to etcd adapter
	var adapterEvents []*adapter.Event
	for _, event := range events {
		// Apply cache changes
		if err := e.applyCacheChange(event); err != nil {
			e.log.Error(err, "failed to apply cache change", "event", event)
			return fmt.Errorf("failed to apply cache change: %w", err)
		}

		// Convert kine event to adapter event
		adapterEvent, err := e.convertToAdapterEvent(event)
		if err != nil {
			e.log.Error(err, "failed to convert event", "event", event)
			return fmt.Errorf("failed to convert event: %w", err)
		}
		adapterEvents = append(adapterEvents, adapterEvent)
	}

	// Send events to etcd adapter
	if len(adapterEvents) > 0 {
		e.log.V(1).Info("sending events to etcd adapter", "count", len(adapterEvents))
		e.adapter.EventCh() <- adapterEvents
		e.log.Info("successfully sent events to etcd adapter")
	} else {
		e.log.Info("no events to send to etcd adapter")
	}

	return nil
}

// convertADCTypesToKineTypes converts ADC resource types to Kine resource types
// ADC Service -> Kine Service + Route
// ADC SSL -> Kine SSL
// ADC GlobalRule -> Kine GlobalRule
func (e *KindExecutor) convertADCTypesToKineTypes(adcTypes []string) []string {
	if len(adcTypes) == 0 {
		// If no types specified, return empty to include all types
		return nil
	}

	kineTypesSet := make(map[string]bool)
	for _, adcType := range adcTypes {
		switch adcType {
		case adctypes.TypeService:
			// ADC Service transfers to both Kine Service and Route
			kineTypesSet[string(kine.ResourceTypeService)] = true
			kineTypesSet[string(kine.ResourceTypeRoute)] = true
		case adctypes.TypeSSL:
			kineTypesSet[string(kine.ResourceTypeSSL)] = true
		case adctypes.TypeGlobalRule:
			kineTypesSet[string(kine.ResourceTypeGlobalRule)] = true
		}
	}

	// Convert set to slice
	kineTypes := make([]string, 0, len(kineTypesSet))
	for kineType := range kineTypesSet {
		kineTypes = append(kineTypes, kineType)
	}

	e.log.V(1).Info("converted ADC types to Kine types", "adcTypes", adcTypes, "kineTypes", kineTypes)
	return kineTypes
}

// applyCacheChange applies a single event to the cache
func (e *KindExecutor) applyCacheChange(event kine.Event) error {
	switch event.Type {
	case kine.EventTypeCreate, kine.EventTypeUpdate:
		return e.cache.Insert(event.NewValue)
	case kine.EventTypeDelete:
		return e.cache.Delete(event.OldValue)
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
}

// convertToAdapterEvent converts a kine event to an adapter event
func (e *KindExecutor) convertToAdapterEvent(event kine.Event) (*adapter.Event, error) {
	// Build key with /apisix prefix
	key := fmt.Sprintf("%s/%s/%s", apisixKeyPrefix, event.ResourceType, event.ResourceID)

	adapterEvent := &adapter.Event{
		Key: key,
	}

	// Set event type
	switch event.Type {
	case kine.EventTypeCreate:
		adapterEvent.Type = adapter.EventAdd
	case kine.EventTypeUpdate:
		adapterEvent.Type = adapter.EventUpdate
	case kine.EventTypeDelete:
		adapterEvent.Type = adapter.EventDelete
	default:
		return nil, fmt.Errorf("unknown event type: %s", event.Type)
	}

	// Serialize value for CREATE and UPDATE events
	if event.Type != kine.EventTypeDelete {
		valueBytes, err := json.Marshal(event.NewValue)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal new value: %w", err)
		}
		adapterEvent.Value = valueBytes
	}

	return adapterEvent, nil
}

// parseArgs parses the command line arguments to extract labels, types, and file path
func (e *KindExecutor) parseArgs(args []string) (map[string]string, []string, string, error) {
	labels := make(map[string]string)
	var types []string
	var filePath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			if i+1 < len(args) {
				filePath = args[i+1]
				i++
			}
		case "--label-selector":
			if i+1 < len(args) {
				labelPair := args[i+1]
				parts := strings.SplitN(labelPair, "=", 2)
				if len(parts) == 2 {
					labels[parts[0]] = parts[1]
				}
				i++
			}
		case "--include-resource-type":
			if i+1 < len(args) {
				types = append(types, args[i+1])
				i++
			}
		}
	}

	if filePath == "" {
		return nil, nil, "", errors.New("file path not found in args")
	}

	return labels, types, filePath, nil
}

// loadResourcesFromFile loads ADC resources from the specified file
func (e *KindExecutor) loadResourcesFromFile(filePath string) (*adctypes.Resources, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var resources adctypes.Resources
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	return &resources, nil
}
