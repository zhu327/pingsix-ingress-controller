package kine

import (
	"testing"

	"github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
)

const (
	testRouteID = "route-1"
)

func TestNewMemDBCache(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	if cache == nil {
		t.Fatal("Cache is nil")
	}
}

func TestCacheRoute(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test route
	route := &Route{
		Metadata: adc.Metadata{
			ID:   testRouteID,
			Name: "test-route",
			Labels: map[string]string{
				label.LabelKind:      "Ingress",
				label.LabelNamespace: "default",
				label.LabelName:      "test",
			},
		},
		URIs:    []string{"/api"},
		Methods: []Method{MethodGET, MethodPOST},
	}

	// Test Insert
	err = cache.InsertRoute(route)
	if err != nil {
		t.Fatalf("Failed to insert route: %v", err)
	}

	// Test Get
	retrieved, err := cache.GetRoute(testRouteID)
	if err != nil {
		t.Fatalf("Failed to get route: %v", err)
	}
	if retrieved.ID != testRouteID {
		t.Errorf("Expected ID %q, got %q", testRouteID, retrieved.ID)
	}
	if retrieved.Name != "test-route" {
		t.Errorf("Expected Name 'test-route', got '%s'", retrieved.Name)
	}

	// Test List
	routes, err := cache.ListRoutes()
	if err != nil {
		t.Fatalf("Failed to list routes: %v", err)
	}
	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}

	// Test Delete
	err = cache.DeleteRoute(route)
	if err != nil {
		t.Fatalf("Failed to delete route: %v", err)
	}

	// Verify deletion
	_, err = cache.GetRoute(testRouteID)
	if err != ErrNotFound {
		t.Error("Expected ErrNotFound after deletion")
	}
}

func TestCacheService(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test service
	service := &Service{
		Metadata: adc.Metadata{
			ID:   "service-1",
			Name: "test-service",
			Labels: map[string]string{
				label.LabelKind:      "Service",
				label.LabelNamespace: "default",
				label.LabelName:      "my-service",
			},
		},
		Hosts: []string{"example.com"},
	}

	// Test Insert
	err = cache.InsertService(service)
	if err != nil {
		t.Fatalf("Failed to insert service: %v", err)
	}

	// Test Get
	retrieved, err := cache.GetService("service-1")
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}
	if retrieved.ID != "service-1" {
		t.Errorf("Expected ID 'service-1', got '%s'", retrieved.ID)
	}

	// Test List
	services, err := cache.ListServices()
	if err != nil {
		t.Fatalf("Failed to list services: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	// Test Delete
	err = cache.DeleteService(service)
	if err != nil {
		t.Fatalf("Failed to delete service: %v", err)
	}
}

func TestCacheUpstream(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test upstream
	upstream := &Upstream{
		Metadata: adc.Metadata{
			ID:   "upstream-1",
			Name: "test-upstream",
		},
		Nodes: map[string]uint32{
			"127.0.0.1:8080": 100,
		},
		Type: SelectionTypeRoundRobin,
	}

	// Test Insert
	err = cache.InsertUpstream(upstream)
	if err != nil {
		t.Fatalf("Failed to insert upstream: %v", err)
	}

	// Test Get
	retrieved, err := cache.GetUpstream("upstream-1")
	if err != nil {
		t.Fatalf("Failed to get upstream: %v", err)
	}
	if retrieved.ID != "upstream-1" {
		t.Errorf("Expected ID 'upstream-1', got '%s'", retrieved.ID)
	}

	// Test List
	upstreams, err := cache.ListUpstreams()
	if err != nil {
		t.Fatalf("Failed to list upstreams: %v", err)
	}
	if len(upstreams) != 1 {
		t.Errorf("Expected 1 upstream, got %d", len(upstreams))
	}

	// Test Delete
	err = cache.DeleteUpstream(upstream)
	if err != nil {
		t.Fatalf("Failed to delete upstream: %v", err)
	}
}

func TestCacheSSL(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test SSL
	ssl := &SSL{
		Metadata: adc.Metadata{
			ID:   "ssl-1",
			Name: "test-ssl",
			Labels: map[string]string{
				label.LabelKind:      "Secret",
				label.LabelNamespace: "default",
				label.LabelName:      "tls-secret",
			},
		},
		Cert: "cert-data",
		Key:  "key-data",
		SNIs: []string{"example.com"},
	}

	// Test Insert
	err = cache.InsertSSL(ssl)
	if err != nil {
		t.Fatalf("Failed to insert SSL: %v", err)
	}

	// Test Get
	retrieved, err := cache.GetSSL("ssl-1")
	if err != nil {
		t.Fatalf("Failed to get SSL: %v", err)
	}
	if retrieved.ID != "ssl-1" {
		t.Errorf("Expected ID 'ssl-1', got '%s'", retrieved.ID)
	}

	// Test List
	ssls, err := cache.ListSSL()
	if err != nil {
		t.Fatalf("Failed to list SSLs: %v", err)
	}
	if len(ssls) != 1 {
		t.Errorf("Expected 1 SSL, got %d", len(ssls))
	}

	// Test Delete
	err = cache.DeleteSSL(ssl)
	if err != nil {
		t.Fatalf("Failed to delete SSL: %v", err)
	}
}

func TestCacheGlobalRule(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test global rule
	globalRule := &GlobalRule{
		ID: "cors",
		Plugins: map[string]any{
			"cors": map[string]any{
				"allow_origins": "**",
			},
		},
	}

	// Test Insert
	err = cache.InsertGlobalRule(globalRule)
	if err != nil {
		t.Fatalf("Failed to insert global rule: %v", err)
	}

	// Test Get
	retrieved, err := cache.GetGlobalRule("cors")
	if err != nil {
		t.Fatalf("Failed to get global rule: %v", err)
	}
	if retrieved.ID != "cors" {
		t.Errorf("Expected ID 'cors', got '%s'", retrieved.ID)
	}

	// Test List
	globalRules, err := cache.ListGlobalRules()
	if err != nil {
		t.Fatalf("Failed to list global rules: %v", err)
	}
	if len(globalRules) != 1 {
		t.Errorf("Expected 1 global rule, got %d", len(globalRules))
	}

	// Test Delete
	err = cache.DeleteGlobalRule(globalRule)
	if err != nil {
		t.Fatalf("Failed to delete global rule: %v", err)
	}
}

func TestCacheListWithLabelSelector(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Insert routes with different labels
	route1 := &Route{
		Metadata: adc.Metadata{
			ID:   testRouteID,
			Name: testRouteID,
			Labels: map[string]string{
				label.LabelKind:      "Ingress",
				label.LabelNamespace: "default",
				label.LabelName:      "ing-1",
			},
		},
		URIs: []string{"/api1"},
	}

	route2 := &Route{
		Metadata: adc.Metadata{
			ID:   "route-2",
			Name: "route-2",
			Labels: map[string]string{
				label.LabelKind:      "Ingress",
				label.LabelNamespace: "default",
				label.LabelName:      "ing-2",
			},
		},
		URIs: []string{"/api2"},
	}

	route3 := &Route{
		Metadata: adc.Metadata{
			ID:   "route-3",
			Name: "route-3",
			Labels: map[string]string{
				label.LabelKind:      "Ingress",
				label.LabelNamespace: "kube-system",
				label.LabelName:      "ing-3",
			},
		},
		URIs: []string{"/api3"},
	}

	if err := cache.InsertRoute(route1); err != nil {
		t.Fatalf("Failed to insert route1: %v", err)
	}
	if err := cache.InsertRoute(route2); err != nil {
		t.Fatalf("Failed to insert route2: %v", err)
	}
	if err := cache.InsertRoute(route3); err != nil {
		t.Fatalf("Failed to insert route3: %v", err)
	}

	// List all routes
	allRoutes, err := cache.ListRoutes()
	if err != nil {
		t.Fatalf("Failed to list all routes: %v", err)
	}
	if len(allRoutes) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(allRoutes))
	}

	// List routes with label selector (default namespace)
	selector := &KindLabelSelector{
		Kind:      "Ingress",
		Namespace: "default",
		Name:      "ing-1",
	}
	filteredRoutes, err := cache.ListRoutes(selector)
	if err != nil {
		t.Fatalf("Failed to list filtered routes: %v", err)
	}
	if len(filteredRoutes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(filteredRoutes))
	}
	if filteredRoutes[0].ID != testRouteID {
		t.Errorf("Expected %s, got %s", testRouteID, filteredRoutes[0].ID)
	}
}

func TestCacheGenericInsertDelete(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test generic Insert
	route := &Route{
		Metadata: adc.Metadata{
			ID:   testRouteID,
			Name: "test-route",
		},
		URIs: []string{"/test"},
	}

	err = cache.Insert(route)
	if err != nil {
		t.Fatalf("Failed to insert via generic Insert: %v", err)
	}

	// Verify insertion
	retrieved, err := cache.GetRoute(testRouteID)
	if err != nil {
		t.Fatalf("Failed to get route: %v", err)
	}
	if retrieved.ID != testRouteID {
		t.Errorf("Expected ID %q, got %q", testRouteID, retrieved.ID)
	}

	// Test generic Delete
	err = cache.Delete(route)
	if err != nil {
		t.Fatalf("Failed to delete via generic Delete: %v", err)
	}

	// Verify deletion
	_, err = cache.GetRoute(testRouteID)
	if err != ErrNotFound {
		t.Error("Expected ErrNotFound after deletion")
	}
}

func TestCacheUpdate(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Insert initial route
	route := &Route{
		Metadata: adc.Metadata{
			ID:   testRouteID,
			Name: "test-route",
		},
		URIs: []string{"/api"},
	}

	err = cache.InsertRoute(route)
	if err != nil {
		t.Fatalf("Failed to insert route: %v", err)
	}

	// Update route (insert with same ID)
	updatedRoute := &Route{
		Metadata: adc.Metadata{
			ID:   testRouteID,
			Name: "updated-route",
		},
		URIs: []string{"/api/v2"},
	}

	err = cache.InsertRoute(updatedRoute)
	if err != nil {
		t.Fatalf("Failed to update route: %v", err)
	}

	// Verify update
	retrieved, err := cache.GetRoute(testRouteID)
	if err != nil {
		t.Fatalf("Failed to get route: %v", err)
	}
	if retrieved.Name != "updated-route" {
		t.Errorf("Expected Name 'updated-route', got %q", retrieved.Name)
	}
	if len(retrieved.URIs) != 1 || retrieved.URIs[0] != "/api/v2" {
		t.Error("Route URIs not updated correctly")
	}
}

func TestCacheDeepCopy(t *testing.T) {
	cache, err := NewMemDBCache()
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a route with complex nested data
	route := &Route{
		Metadata: adc.Metadata{
			ID:   "route-1",
			Name: "test-route",
			Labels: map[string]string{
				"key": "value",
			},
		},
		URIs:    []string{"/api"},
		Methods: []Method{MethodGET},
		Plugins: map[string]any{
			"cors": map[string]any{
				"enabled": true,
			},
		},
	}

	err = cache.InsertRoute(route)
	if err != nil {
		t.Fatalf("Failed to insert route: %v", err)
	}

	// Get route from cache
	retrieved, err := cache.GetRoute("route-1")
	if err != nil {
		t.Fatalf("Failed to get route: %v", err)
	}

	// Modify retrieved copy
	retrieved.Name = "modified"
	retrieved.URIs[0] = "/modified"

	// Get route again and verify original is unchanged
	retrieved2, err := cache.GetRoute("route-1")
	if err != nil {
		t.Fatalf("Failed to get route again: %v", err)
	}

	if retrieved2.Name != "test-route" {
		t.Error("Original route was modified (deep copy failed)")
	}
	if retrieved2.URIs[0] != "/api" {
		t.Error("Original route URIs were modified (deep copy failed)")
	}
}
