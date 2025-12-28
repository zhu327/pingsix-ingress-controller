package kine

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/apache/apisix-ingress-controller/api/adc"
)

const (
	exampleHost = "example.com"
	upstream1   = "upstream1"
)

func TestDiffer_DiffRoutes(t *testing.T) {
	// Create cache
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Insert existing routes into cache
	existingRoute := &Route{
		Metadata: adc.Metadata{
			ID:   "route1",
			Name: "existing-route",
			Labels: map[string]string{
				"k8s/kind":      "ApisixRoute",
				"k8s/namespace": "default",
				"k8s/name":      "test",
			},
		},
		URIs: []string{"/test"},
	}
	if err := cache.InsertRoute(existingRoute); err != nil {
		t.Fatalf("failed to insert route: %v", err)
	}

	// Create differ
	differ := NewDiffer(cache)

	// New resources: one route to update, one new route, existing route is deleted
	newResources := &TransferredResources{
		Routes: []*Route{
			{
				Metadata: adc.Metadata{
					ID:   "route1",
					Name: "existing-route",
					Labels: map[string]string{
						"k8s/kind":      "ApisixRoute",
						"k8s/namespace": "default",
						"k8s/name":      "test",
					},
				},
				URIs: []string{"/test", "/test2"}, // Modified
			},
			{
				Metadata: adc.Metadata{
					ID:   "route2",
					Name: "new-route",
					Labels: map[string]string{
						"k8s/kind":      "ApisixRoute",
						"k8s/namespace": "default",
						"k8s/name":      "test",
					},
				},
				URIs: []string{"/new"},
			},
		},
	}

	// Perform diff
	opts := &DiffOptions{
		Labels: map[string]string{
			"k8s/kind":      "ApisixRoute",
			"k8s/namespace": "default",
			"k8s/name":      "test",
		},
	}

	events, err := differ.Diff(newResources, opts)
	if err != nil {
		t.Fatalf("failed to diff: %v", err)
	}

	// Verify events
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	// Find update and create events
	var hasUpdate, hasCreate bool
	for _, event := range events {
		if event.Type == EventTypeUpdate && event.ResourceID == "route1" {
			hasUpdate = true
		}
		if event.Type == EventTypeCreate && event.ResourceID == "route2" {
			hasCreate = true
		}
	}

	if !hasUpdate {
		t.Error("expected UPDATE event for route1")
	}
	if !hasCreate {
		t.Error("expected CREATE event for route2")
	}
}

func TestDiffer_DiffServices(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Insert existing service
	existingService := &Service{
		Metadata: adc.Metadata{
			ID:   "service1",
			Name: "existing-service",
			Labels: map[string]string{
				"k8s/kind":      "Service",
				"k8s/namespace": "default",
				"k8s/name":      "test",
			},
		},
		Hosts: []string{exampleHost},
	}
	if err := cache.InsertService(existingService); err != nil {
		t.Fatalf("failed to insert service: %v", err)
	}

	differ := NewDiffer(cache)

	// New resources: delete existing service
	newResources := &TransferredResources{
		Services: []*Service{},
	}

	opts := &DiffOptions{
		Labels: map[string]string{
			"k8s/kind":      "Service",
			"k8s/namespace": "default",
			"k8s/name":      "test",
		},
	}

	events, err := differ.Diff(newResources, opts)
	if err != nil {
		t.Fatalf("failed to diff: %v", err)
	}

	// Should have one DELETE event
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != EventTypeDelete {
		t.Errorf("expected DELETE event, got %v", events[0].Type)
	}
	if events[0].ResourceID != "service1" {
		t.Errorf("expected resource ID service1, got %v", events[0].ResourceID)
	}
}

func TestDiffer_DiffUpstreams(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Insert existing upstream
	existingUpstream := &Upstream{
		Metadata: adc.Metadata{
			ID:   "upstream1",
			Name: "existing-upstream",
			Labels: map[string]string{
				"k8s/kind":      "Upstream",
				"k8s/namespace": "default",
				"k8s/name":      "test",
			},
		},
		Nodes: map[string]uint32{
			"127.0.0.1:8080": 100,
		},
		Type: SelectionTypeRoundRobin,
	}
	if err := cache.InsertUpstream(existingUpstream); err != nil {
		t.Fatalf("failed to insert upstream: %v", err)
	}

	differ := NewDiffer(cache)

	// New resources: update existing upstream, create new upstream
	newResources := &TransferredResources{
		Upstreams: []*Upstream{
			{
				Metadata: adc.Metadata{
					ID:   "upstream1",
					Name: "existing-upstream",
					Labels: map[string]string{
						"k8s/kind":      "Upstream",
						"k8s/namespace": "default",
						"k8s/name":      "test",
					},
				},
				Nodes: map[string]uint32{
					"127.0.0.1:8080": 100,
					"127.0.0.2:8080": 50, // Modified: added node
				},
				Type: SelectionTypeRoundRobin,
			},
			{
				Metadata: adc.Metadata{
					ID:   "upstream2",
					Name: "new-upstream",
					Labels: map[string]string{
						"k8s/kind":      "Upstream",
						"k8s/namespace": "default",
						"k8s/name":      "test",
					},
				},
				Nodes: map[string]uint32{
					"192.168.1.1:9090": 100,
				},
				Type: SelectionTypeRandom,
			},
		},
	}

	opts := &DiffOptions{
		Labels: map[string]string{
			"k8s/kind":      "Upstream",
			"k8s/namespace": "default",
			"k8s/name":      "test",
		},
	}

	events, err := differ.Diff(newResources, opts)
	if err != nil {
		t.Fatalf("failed to diff: %v", err)
	}

	// Should have one UPDATE and one CREATE event
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	var hasUpdate, hasCreate bool
	for _, event := range events {
		if event.Type == EventTypeUpdate && event.ResourceID == upstream1 {
			hasUpdate = true
		}
		if event.Type == EventTypeCreate && event.ResourceID == "upstream2" {
			hasCreate = true
		}
	}

	if !hasUpdate {
		t.Error("expected UPDATE event for " + upstream1)
	}
	if !hasCreate {
		t.Error("expected CREATE event for upstream2")
	}
}

func TestDiffer_DiffUpstreamsDelete(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Insert existing upstream
	existingUpstream := &Upstream{
		Metadata: adc.Metadata{
			ID:   "upstream1",
			Name: "test-upstream",
			Labels: map[string]string{
				"k8s/kind":      "Upstream",
				"k8s/namespace": "default",
				"k8s/name":      "test",
			},
		},
		Nodes: map[string]uint32{
			"127.0.0.1:8080": 100,
		},
		Type: SelectionTypeRoundRobin,
	}
	if err := cache.InsertUpstream(existingUpstream); err != nil {
		t.Fatalf("failed to insert upstream: %v", err)
	}

	differ := NewDiffer(cache)

	// New resources: empty (delete existing upstream)
	newResources := &TransferredResources{
		Upstreams: []*Upstream{},
	}

	opts := &DiffOptions{
		Labels: map[string]string{
			"k8s/kind":      "Upstream",
			"k8s/namespace": "default",
			"k8s/name":      "test",
		},
	}

	events, err := differ.Diff(newResources, opts)
	if err != nil {
		t.Fatalf("failed to diff: %v", err)
	}

	// Should have one DELETE event
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != EventTypeDelete {
		t.Errorf("expected DELETE event, got %v", events[0].Type)
	}
	if events[0].ResourceID != upstream1 {
		t.Errorf("expected resource ID %s, got %v", upstream1, events[0].ResourceID)
	}
	if events[0].ResourceType != ResourceTypeUpstream {
		t.Errorf("expected resource type upstream, got %v", events[0].ResourceType)
	}
}

func TestSortEvents(t *testing.T) {
	events := []Event{
		{Type: EventTypeCreate, ResourceType: ResourceTypeRoute},
		{Type: EventTypeDelete, ResourceType: ResourceTypeService},
		{Type: EventTypeUpdate, ResourceType: ResourceTypeSSL},
		{Type: EventTypeCreate, ResourceType: ResourceTypeService},
		{Type: EventTypeDelete, ResourceType: ResourceTypeRoute},
		{Type: EventTypeCreate, ResourceType: ResourceTypeSSL},
	}

	sortEvents(events)

	// Verify order
	// DELETE events first (Route -> Service -> SSL)
	if events[0].Type != EventTypeDelete || events[0].ResourceType != ResourceTypeRoute {
		t.Errorf("expected DELETE Route first, got %v %v", events[0].Type, events[0].ResourceType)
	}
	if events[1].Type != EventTypeDelete || events[1].ResourceType != ResourceTypeService {
		t.Errorf("expected DELETE Service second, got %v %v", events[1].Type, events[1].ResourceType)
	}

	// UPDATE events in the middle
	if events[2].Type != EventTypeUpdate {
		t.Errorf("expected UPDATE event in middle, got %v", events[2].Type)
	}

	// CREATE events last (SSL -> Service -> Route)
	if events[3].Type != EventTypeCreate || events[3].ResourceType != ResourceTypeSSL {
		t.Errorf("expected CREATE SSL, got %v %v", events[3].Type, events[3].ResourceType)
	}
	if events[4].Type != EventTypeCreate || events[4].ResourceType != ResourceTypeService {
		t.Errorf("expected CREATE Service, got %v %v", events[4].Type, events[4].ResourceType)
	}
	if events[5].Type != EventTypeCreate || events[5].ResourceType != ResourceTypeRoute {
		t.Errorf("expected CREATE Route last, got %v %v", events[5].Type, events[5].ResourceType)
	}
}

func TestSortEventsWithUpstream(t *testing.T) {
	events := []Event{
		{Type: EventTypeCreate, ResourceType: ResourceTypeRoute},
		{Type: EventTypeCreate, ResourceType: ResourceTypeService},
		{Type: EventTypeCreate, ResourceType: ResourceTypeUpstream},
		{Type: EventTypeCreate, ResourceType: ResourceTypeSSL},
		{Type: EventTypeCreate, ResourceType: ResourceTypeGlobalRule},
		{Type: EventTypeDelete, ResourceType: ResourceTypeRoute},
		{Type: EventTypeDelete, ResourceType: ResourceTypeService},
		{Type: EventTypeDelete, ResourceType: ResourceTypeUpstream},
		{Type: EventTypeDelete, ResourceType: ResourceTypeSSL},
		{Type: EventTypeDelete, ResourceType: ResourceTypeGlobalRule},
		{Type: EventTypeUpdate, ResourceType: ResourceTypeRoute},
		{Type: EventTypeUpdate, ResourceType: ResourceTypeUpstream},
	}

	sortEvents(events)

	// Verify DELETE events order: Route -> Service -> Upstream -> SSL -> GlobalRule
	if events[0].Type != EventTypeDelete || events[0].ResourceType != ResourceTypeRoute {
		t.Errorf("expected DELETE Route first, got %v %v", events[0].Type, events[0].ResourceType)
	}
	if events[1].Type != EventTypeDelete || events[1].ResourceType != ResourceTypeService {
		t.Errorf("expected DELETE Service, got %v %v", events[1].Type, events[1].ResourceType)
	}
	if events[2].Type != EventTypeDelete || events[2].ResourceType != ResourceTypeUpstream {
		t.Errorf("expected DELETE Upstream, got %v %v", events[2].Type, events[2].ResourceType)
	}
	if events[3].Type != EventTypeDelete || events[3].ResourceType != ResourceTypeSSL {
		t.Errorf("expected DELETE SSL, got %v %v", events[3].Type, events[3].ResourceType)
	}
	if events[4].Type != EventTypeDelete || events[4].ResourceType != ResourceTypeGlobalRule {
		t.Errorf("expected DELETE GlobalRule, got %v %v", events[4].Type, events[4].ResourceType)
	}

	// Verify UPDATE events order: Route -> Service -> Upstream (same as DELETE)
	if events[5].Type != EventTypeUpdate || events[5].ResourceType != ResourceTypeRoute {
		t.Errorf("expected UPDATE Route, got %v %v", events[5].Type, events[5].ResourceType)
	}
	if events[6].Type != EventTypeUpdate || events[6].ResourceType != ResourceTypeUpstream {
		t.Errorf("expected UPDATE Upstream, got %v %v", events[6].Type, events[6].ResourceType)
	}

	// Verify CREATE events order: GlobalRule -> SSL -> Upstream -> Service -> Route
	if events[7].Type != EventTypeCreate || events[7].ResourceType != ResourceTypeGlobalRule {
		t.Errorf("expected CREATE GlobalRule, got %v %v", events[7].Type, events[7].ResourceType)
	}
	if events[8].Type != EventTypeCreate || events[8].ResourceType != ResourceTypeSSL {
		t.Errorf("expected CREATE SSL, got %v %v", events[8].Type, events[8].ResourceType)
	}
	if events[9].Type != EventTypeCreate || events[9].ResourceType != ResourceTypeUpstream {
		t.Errorf("expected CREATE Upstream, got %v %v", events[9].Type, events[9].ResourceType)
	}
	if events[10].Type != EventTypeCreate || events[10].ResourceType != ResourceTypeService {
		t.Errorf("expected CREATE Service, got %v %v", events[10].Type, events[10].ResourceType)
	}
	if events[11].Type != EventTypeCreate || events[11].ResourceType != ResourceTypeRoute {
		t.Errorf("expected CREATE Route last, got %v %v", events[11].Type, events[11].ResourceType)
	}
}

func TestTransferResources(t *testing.T) {
	// Create ADC resources
	host := exampleHost
	resources := &adc.Resources{
		Services: []*adc.Service{
			{
				Metadata: adc.Metadata{
					ID:   "service1",
					Name: "test-service",
					Labels: map[string]string{
						"k8s/kind":      "Service",
						"k8s/namespace": "default",
						"k8s/name":      "test",
					},
				},
				Hosts: []string{host},
				Upstream: &adc.Upstream{
					Metadata: adc.Metadata{
						Name: "test-upstream",
					},
					Nodes: []adc.UpstreamNode{
						{
							Host:   "127.0.0.1",
							Port:   8080,
							Weight: 100,
						},
					},
					Type: adc.Roundrobin,
				},
				Routes: []*adc.Route{
					{
						Metadata: adc.Metadata{
							Name: "test-route",
						},
						Uris: []string{"/test"},
					},
				},
			},
		},
		SSLs: []*adc.SSL{
			{
				Metadata: adc.Metadata{
					Name: "test-ssl",
				},
				Certificates: []adc.Certificate{
					{
						Certificate: "cert1",
						Key:         "key1",
					},
				},
				Snis: []string{exampleHost},
			},
		},
		GlobalRules: adc.GlobalRule{
			"limit-req": map[string]any{
				"rate":  100,
				"burst": 200,
			},
		},
	}

	// Transfer resources
	transferred, err := TransferResources(resources)
	if err != nil {
		t.Fatalf("failed to transfer resources: %v", err)
	}

	// Verify transferred resources
	if len(transferred.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(transferred.Services))
	}
	if len(transferred.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(transferred.Routes))
	}
	if len(transferred.SSLs) != 1 {
		t.Errorf("expected 1 ssl, got %d", len(transferred.SSLs))
	}
	if len(transferred.GlobalRules) != 1 {
		t.Errorf("expected 1 global rule, got %d", len(transferred.GlobalRules))
	}

	// Verify that service contains upstream
	if transferred.Services[0].Upstream == nil {
		t.Error("expected service to contain upstream")
	}
}

func TestTransferResourcesWithUpstreams(t *testing.T) {
	// Create ADC resources with Upstreams field
	resources := &adc.Resources{
		Services: []*adc.Service{
			{
				Metadata: adc.Metadata{
					Name: "test-service",
				},
				Upstream: &adc.Upstream{
					Metadata: adc.Metadata{
						Name: "default-upstream",
					},
					Nodes: []adc.UpstreamNode{
						{Host: "127.0.0.1", Port: 8080, Weight: 100},
					},
					Type: adc.Roundrobin,
				},
				Upstreams: []*adc.Upstream{
					{
						Metadata: adc.Metadata{
							ID:   "upstream1",
							Name: "named-upstream-1",
						},
						Nodes: []adc.UpstreamNode{
							{Host: "192.168.1.1", Port: 8080, Weight: 100},
						},
						Type: adc.Roundrobin,
					},
					{
						Metadata: adc.Metadata{
							Name: "named-upstream-2",
						},
						Nodes: []adc.UpstreamNode{
							{Host: "10.0.0.1", Port: 9090, Weight: 80},
						},
						Type: adc.Random,
					},
				},
				Routes: []*adc.Route{
					{
						Metadata: adc.Metadata{
							Name: "route1",
						},
						Uris: []string{"/api"},
					},
				},
			},
		},
	}

	// Transfer resources
	transferred, err := TransferResources(resources)
	if err != nil {
		t.Fatalf("failed to transfer resources: %v", err)
	}

	// Verify upstreams were transferred
	if len(transferred.Upstreams) != 2 {
		t.Errorf("expected 2 upstreams, got %d", len(transferred.Upstreams))
	}

	// Verify first upstream
	if transferred.Upstreams[0].ID != "upstream1" {
		t.Errorf("expected upstream ID 'upstream1', got '%s'", transferred.Upstreams[0].ID)
	}
	if transferred.Upstreams[0].Name != "named-upstream-1" {
		t.Errorf("expected upstream name 'named-upstream-1', got '%s'", transferred.Upstreams[0].Name)
	}

	// Verify second upstream (ID should be generated)
	if transferred.Upstreams[1].Name != "named-upstream-2" {
		t.Errorf("expected upstream name 'named-upstream-2', got '%s'", transferred.Upstreams[1].Name)
	}
}

func TestTransferResourcesMultipleServices(t *testing.T) {
	// Create ADC resources with multiple services, each having upstreams
	resources := &adc.Resources{
		Services: []*adc.Service{
			{
				Metadata: adc.Metadata{
					Name: "service1",
				},
				Upstream: &adc.Upstream{
					Metadata: adc.Metadata{
						Name: "upstream1",
					},
					Nodes: []adc.UpstreamNode{
						{Host: "127.0.0.1", Port: 8080, Weight: 100},
					},
					Type: adc.Roundrobin,
				},
				Upstreams: []*adc.Upstream{
					{
						Metadata: adc.Metadata{
							Name: "named-upstream-1",
						},
						Nodes: []adc.UpstreamNode{
							{Host: "192.168.1.1", Port: 8080, Weight: 100},
						},
						Type: adc.Roundrobin,
					},
				},
				Routes: []*adc.Route{
					{
						Metadata: adc.Metadata{
							Name: "route1",
						},
						Uris: []string{"/api1"},
					},
				},
			},
			{
				Metadata: adc.Metadata{
					Name: "service2",
				},
				Upstream: &adc.Upstream{
					Metadata: adc.Metadata{
						Name: "upstream2",
					},
					Nodes: []adc.UpstreamNode{
						{Host: "127.0.0.2", Port: 9090, Weight: 50},
					},
					Type: adc.Random,
				},
				Upstreams: []*adc.Upstream{
					{
						Metadata: adc.Metadata{
							Name: "named-upstream-2",
						},
						Nodes: []adc.UpstreamNode{
							{Host: "10.0.0.1", Port: 9090, Weight: 80},
						},
						Type: adc.Chash,
					},
				},
				Routes: []*adc.Route{
					{
						Metadata: adc.Metadata{
							Name: "route2",
						},
						Uris: []string{"/api2"},
					},
				},
			},
		},
	}

	// Transfer resources
	transferred, err := TransferResources(resources)
	if err != nil {
		t.Fatalf("failed to transfer resources: %v", err)
	}

	// Verify services
	if len(transferred.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(transferred.Services))
	}

	// Verify routes
	if len(transferred.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(transferred.Routes))
	}

	// Verify upstreams - should have 2 (one from each service's Upstreams field)
	if len(transferred.Upstreams) != 2 {
		t.Errorf("expected 2 upstreams, got %d", len(transferred.Upstreams))
	}

	// Verify upstream names
	upstreamNames := make(map[string]bool)
	for _, upstream := range transferred.Upstreams {
		upstreamNames[upstream.Name] = true
	}

	if !upstreamNames["named-upstream-1"] {
		t.Error("expected to find named-upstream-1")
	}
	if !upstreamNames["named-upstream-2"] {
		t.Error("expected to find named-upstream-2")
	}
}

func TestCmpEqual(t *testing.T) {
	route1 := &Route{
		Metadata: adc.Metadata{
			ID:   "route1",
			Name: "test",
		},
		URIs: []string{"/test"},
	}

	route2 := &Route{
		Metadata: adc.Metadata{
			ID:   "route1",
			Name: "test",
		},
		URIs: []string{"/test"},
	}

	route3 := &Route{
		Metadata: adc.Metadata{
			ID:   "route1",
			Name: "test",
		},
		URIs: []string{"/test", "/test2"},
	}

	if !cmp.Equal(route1, route2) {
		t.Error("expected route1 and route2 to be equal")
	}

	if cmp.Equal(route1, route3) {
		t.Error("expected route1 and route3 to be different")
	}
}
