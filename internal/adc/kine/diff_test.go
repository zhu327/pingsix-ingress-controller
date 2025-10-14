package kine

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/apache/apisix-ingress-controller/api/adc"
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
		KindSelector: &KindLabelSelector{
			Kind:      "ApisixRoute",
			Namespace: "default",
			Name:      "test",
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
		Hosts: []string{"example.com"},
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
		KindSelector: &KindLabelSelector{
			Kind:      "Service",
			Namespace: "default",
			Name:      "test",
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

func TestTransferResources(t *testing.T) {
	// Create ADC resources
	host := "example.com"
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
				Snis: []string{"example.com"},
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
