package kine

import (
	"fmt"
	"testing"

	"github.com/apache/apisix-ingress-controller/api/adc"
)

func TestTransferService(t *testing.T) {
	// Create a test ADC Service
	priority := int64(10)
	adcSvc := &adc.Service{
		Metadata: adc.Metadata{
			ID:   "",
			Name: "test-service",
			Desc: "Test service description",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Hosts: []string{"example.com"},
		Plugins: adc.Plugins{
			"cors": map[string]any{
				"allow_origins": "**",
			},
		},
		Upstream: &adc.Upstream{
			Metadata: adc.Metadata{
				ID:   "upstream-1",
				Name: "test-upstream",
			},
			Nodes: adc.UpstreamNodes{
				{Host: "127.0.0.1", Port: 8080, Weight: 100},
				{Host: "127.0.0.2", Port: 8080, Weight: 50},
			},
			Type:     adc.Roundrobin,
			Scheme:   "http",
			PassHost: "pass",
		},
		Routes: []*adc.Route{
			{
				Metadata: adc.Metadata{
					ID:   "",
					Name: "route1",
					Desc: "Test route 1",
				},
				Uris:     []string{"/api/v1"},
				Methods:  []string{"GET", "POST"},
				Hosts:    []string{"example.com"},
				Priority: &priority,
				Plugins: adc.Plugins{
					"rate-limit": map[string]any{
						"rate": 100,
					},
				},
			},
			{
				Metadata: adc.Metadata{
					ID:   "custom-route-id",
					Name: "route2",
				},
				Uris:    []string{"/api/v2"},
				Methods: []string{"GET"},
			},
		},
	}

	// Test TransferService
	kineSvc, kineRoutes, err := TransferService(adcSvc)
	if err != nil {
		t.Fatalf("TransferService failed: %v", err)
	}

	// Verify Service
	if kineSvc == nil {
		t.Fatal("kineSvc is nil")
	}

	// Check if ID is generated when not provided
	if kineSvc.ID == "" {
		t.Error("Service ID should be generated")
	}

	if kineSvc.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", kineSvc.Name)
	}

	if len(kineSvc.Hosts) != 1 || kineSvc.Hosts[0] != "example.com" {
		t.Errorf("Service hosts mismatch")
	}

	// Verify Upstream
	if kineSvc.Upstream == nil {
		t.Fatal("Service upstream is nil")
	}

	if len(kineSvc.Upstream.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(kineSvc.Upstream.Nodes))
	}

	if kineSvc.Upstream.Type != SelectionTypeRoundRobin {
		t.Errorf("Expected type roundrobin, got %s", kineSvc.Upstream.Type)
	}

	// Verify Routes
	if len(kineRoutes) != 2 {
		t.Fatalf("Expected 2 routes, got %d", len(kineRoutes))
	}

	// Check first route (ID should be generated)
	route1 := kineRoutes[0]
	if route1.ID == "" {
		t.Error("Route1 ID should be generated")
	}

	expectedRoute1ID := "4e8b8c7410909de7e7fcd863ed3065260421306a" // sha1("test-service.route1")
	if route1.ID != expectedRoute1ID {
		t.Errorf("Route1 ID mismatch. Expected %s, got %s", expectedRoute1ID, route1.ID)
	}

	if route1.Name != "route1" {
		t.Errorf("Expected route name 'route1', got '%s'", route1.Name)
	}

	if len(route1.URIs) != 1 || route1.URIs[0] != "/api/v1" {
		t.Error("Route1 URIs mismatch")
	}

	if len(route1.Methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(route1.Methods))
	}

	if route1.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", route1.Priority)
	}

	if route1.ServiceID == nil {
		t.Error("Route1 ServiceID should not be nil")
	}

	// Check second route (ID is provided)
	route2 := kineRoutes[1]
	if route2.ID != "custom-route-id" {
		t.Errorf("Expected route ID 'custom-route-id', got '%s'", route2.ID)
	}

	if route2.Name != "route2" {
		t.Errorf("Expected route name 'route2', got '%s'", route2.Name)
	}

	if route2.ServiceID == nil {
		t.Error("Route2 ServiceID should not be nil")
	}
}

func TestTransferServiceWithCustomID(t *testing.T) {
	adcSvc := &adc.Service{
		Metadata: adc.Metadata{
			ID:   "custom-service-id",
			Name: "test-service",
		},
		Upstream: &adc.Upstream{
			Nodes: adc.UpstreamNodes{
				{Host: "127.0.0.1", Port: 8080, Weight: 100},
			},
		},
		Routes: []*adc.Route{
			{
				Metadata: adc.Metadata{
					Name: "route1",
				},
				Uris: []string{"/test"},
			},
		},
	}

	kineSvc, kineRoutes, err := TransferService(adcSvc)
	if err != nil {
		t.Fatalf("TransferService failed: %v", err)
	}

	// Service should use the provided ID
	if kineSvc.ID != "custom-service-id" {
		t.Errorf("Expected service ID 'custom-service-id', got '%s'", kineSvc.ID)
	}

	// Route should reference the service ID
	if kineRoutes[0].ServiceID == nil || *kineRoutes[0].ServiceID != "custom-service-id" {
		t.Error("Route should reference custom-service-id")
	}
}

func TestTransferServiceNilUpstream(t *testing.T) {
	adcSvc := &adc.Service{
		Metadata: adc.Metadata{
			Name: "test-service",
		},
		Upstream: nil,
	}

	_, _, err := TransferService(adcSvc)
	if err == nil {
		t.Error("Expected error for nil upstream")
	}
}

func TestSha1Hash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test", "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"},
		{"test-service", "b3f5226339e8021d693aa1127467a2e7c5dfb012"},
		{"test-service.route1", "4e8b8c7410909de7e7fcd863ed3065260421306a"},
	}

	for _, tt := range tests {
		result := sha1Hash(tt.input)
		if result != tt.expected {
			t.Errorf("sha1Hash(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestConvertUpstreamType(t *testing.T) {
	tests := []struct {
		input    adc.UpstreamType
		expected SelectionType
	}{
		{adc.Roundrobin, SelectionTypeRoundRobin},
		{adc.Random, SelectionTypeRandom},
		{adc.Chash, SelectionTypeFnv},
		{adc.Ketama, SelectionTypeKetama},
		{adc.LeastConn, SelectionTypeRoundRobin},
		{adc.Ewma, SelectionTypeRoundRobin},
	}

	for _, tt := range tests {
		result := convertUpstreamType(tt.input)
		if result != tt.expected {
			t.Errorf("convertUpstreamType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestConvertNodes(t *testing.T) {
	nodes := adc.UpstreamNodes{
		{Host: "127.0.0.1", Port: 8080, Weight: 100},
		{Host: "192.168.1.1", Port: 9090, Weight: 50},
	}

	result := convertNodes(nodes)

	if len(result) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(result))
	}

	if result["127.0.0.1:8080"] != 100 {
		t.Errorf("Expected weight 100 for 127.0.0.1:8080, got %d", result["127.0.0.1:8080"])
	}

	if result["192.168.1.1:9090"] != 50 {
		t.Errorf("Expected weight 50 for 192.168.1.1:9090, got %d", result["192.168.1.1:9090"])
	}
}

func TestConvertMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT"}
	result := convertMethods(methods)

	if len(result) != 3 {
		t.Fatalf("Expected 3 methods, got %d", len(result))
	}

	expected := []Method{MethodGET, MethodPOST, MethodPUT}
	for i, m := range result {
		if m != expected[i] {
			t.Errorf("Expected method %s at index %d, got %s", expected[i], i, m)
		}
	}
}

func TestConvertTimeout(t *testing.T) {
	adcTimeout := &adc.Timeout{
		Connect: 10,
		Send:    20,
		Read:    30,
	}

	result := convertTimeout(adcTimeout)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Connect != 10 || result.Send != 20 || result.Read != 30 {
		t.Error("Timeout values mismatch")
	}

	// Test nil timeout
	nilResult := convertTimeout(nil)
	if nilResult != nil {
		t.Error("Expected nil for nil input")
	}
}

func TestConvertScheme(t *testing.T) {
	tests := []struct {
		input    string
		expected UpstreamScheme
	}{
		{"http", UpstreamSchemeHTTP},
		{"https", UpstreamSchemeHTTPS},
		{"grpc", UpstreamSchemeGRPC},
		{"grpcs", UpstreamSchemeGRPCS},
		{"unknown", UpstreamSchemeHTTP}, // default
	}

	for _, tt := range tests {
		result := convertScheme(tt.input)
		if result != tt.expected {
			t.Errorf("convertScheme(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestConvertPassHost(t *testing.T) {
	tests := []struct {
		input    string
		expected UpstreamPassHost
	}{
		{"pass", UpstreamPassHostPass},
		{"rewrite", UpstreamPassHostRewrite},
		{"unknown", UpstreamPassHostPass}, // default
	}

	for _, tt := range tests {
		result := convertPassHost(tt.input)
		if result != tt.expected {
			t.Errorf("convertPassHost(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestConvertUpstreamWithoutID(t *testing.T) {
	// Test upstream without ID - should generate one from name
	adcUpstream := &adc.Upstream{
		Metadata: adc.Metadata{
			ID:   "", // No ID provided
			Name: "test-upstream",
		},
		Nodes: adc.UpstreamNodes{
			{Host: "127.0.0.1", Port: 8080, Weight: 100},
		},
		Type:   adc.Roundrobin,
		Scheme: "http",
	}

	result := convertUpstream(adcUpstream)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	expectedID := sha1Hash("test-upstream")
	if result.ID != expectedID {
		t.Errorf("Expected upstream ID to be generated as %s, got %s", expectedID, result.ID)
	}

	if result.Name != "test-upstream" {
		t.Errorf("Expected name 'test-upstream', got '%s'", result.Name)
	}
}

func TestConvertUpstreamWithID(t *testing.T) {
	// Test upstream with ID - should use the provided ID
	adcUpstream := &adc.Upstream{
		Metadata: adc.Metadata{
			ID:   "custom-upstream-id",
			Name: "test-upstream",
		},
		Nodes: adc.UpstreamNodes{
			{Host: "127.0.0.1", Port: 8080, Weight: 100},
		},
		Type:   adc.Roundrobin,
		Scheme: "http",
	}

	result := convertUpstream(adcUpstream)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.ID != "custom-upstream-id" {
		t.Errorf("Expected upstream ID 'custom-upstream-id', got '%s'", result.ID)
	}

	if result.Name != "test-upstream" {
		t.Errorf("Expected name 'test-upstream', got '%s'", result.Name)
	}
}

func TestConvertUpstreamWithHealthCheck(t *testing.T) {
	// Test upstream with health check
	adcUpstream := &adc.Upstream{
		Metadata: adc.Metadata{
			Name: "test-upstream",
		},
		Nodes: adc.UpstreamNodes{
			{Host: "127.0.0.1", Port: 8080, Weight: 100},
		},
		Type: adc.Roundrobin,
		Checks: &adc.UpstreamHealthCheck{
			Active: &adc.UpstreamActiveHealthCheck{
				Type:     "http",
				Timeout:  5,
				HTTPPath: "/health",
				Host:     "example.com",
				Port:     8080,
				Healthy: adc.UpstreamActiveHealthCheckHealthy{
					Interval: 10,
					UpstreamPassiveHealthCheckHealthy: adc.UpstreamPassiveHealthCheckHealthy{
						HTTPStatuses: []int{200, 201},
						Successes:    3,
					},
				},
				Unhealthy: adc.UpstreamActiveHealthCheckUnhealthy{
					Interval: 5,
					UpstreamPassiveHealthCheckUnhealthy: adc.UpstreamPassiveHealthCheckUnhealthy{
						HTTPFailures: 5,
						TCPFailures:  2,
					},
				},
			},
		},
	}

	result := convertUpstream(adcUpstream)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Checks == nil {
		t.Fatal("Health check should not be nil")
	}

	if result.Checks.Active == nil {
		t.Fatal("Active health check should not be nil")
	}

	if result.Checks.Active.Type != ActiveCheckTypeHTTP {
		t.Errorf("Expected check type HTTP, got %s", result.Checks.Active.Type)
	}

	if result.Checks.Active.HTTPPath != "/health" {
		t.Errorf("Expected HTTP path '/health', got '%s'", result.Checks.Active.HTTPPath)
	}

	if result.Checks.Active.Timeout != 5 {
		t.Errorf("Expected timeout 5, got %d", result.Checks.Active.Timeout)
	}

	if result.Checks.Active.Healthy == nil {
		t.Fatal("Healthy check should not be nil")
	}

	if result.Checks.Active.Healthy.Interval != 10 {
		t.Errorf("Expected healthy interval 10, got %d", result.Checks.Active.Healthy.Interval)
	}

	if len(result.Checks.Active.Healthy.HTTPStatuses) != 2 {
		t.Errorf("Expected 2 HTTP statuses, got %d", len(result.Checks.Active.Healthy.HTTPStatuses))
	}

	if result.Checks.Active.Unhealthy == nil {
		t.Fatal("Unhealthy check should not be nil")
	}

	if result.Checks.Active.Unhealthy.HTTPFailures != 5 {
		t.Errorf("Expected HTTP failures 5, got %d", result.Checks.Active.Unhealthy.HTTPFailures)
	}

	if result.Checks.Active.Unhealthy.TCPFailures != 2 {
		t.Errorf("Expected TCP failures 2, got %d", result.Checks.Active.Unhealthy.TCPFailures)
	}
}

func TestTransferSSLSingleCertificateWithID(t *testing.T) {
	// Test SSL with single certificate and custom ID
	adcSSL := &adc.SSL{
		Metadata: adc.Metadata{
			ID:   "custom-ssl-id",
			Name: "test-ssl",
			Desc: "Test SSL certificate",
			Labels: map[string]string{
				"env": "prod",
			},
		},
		Certificates: []adc.Certificate{
			{
				Certificate: "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----",
				Key:         "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----",
			},
		},
		Snis: []string{"example.com", "*.example.com"},
	}

	kineSSLs, err := TransferSSL(adcSSL)
	if err != nil {
		t.Fatalf("TransferSSL failed: %v", err)
	}

	if len(kineSSLs) != 1 {
		t.Fatalf("Expected 1 SSL, got %d", len(kineSSLs))
	}

	kineSSL := kineSSLs[0]

	// Should use the provided ID
	if kineSSL.ID != "custom-ssl-id" {
		t.Errorf("Expected SSL ID 'custom-ssl-id', got '%s'", kineSSL.ID)
	}

	if kineSSL.Name != "test-ssl" {
		t.Errorf("Expected name 'test-ssl', got '%s'", kineSSL.Name)
	}

	if kineSSL.Desc != "Test SSL certificate" {
		t.Errorf("Expected desc 'Test SSL certificate', got '%s'", kineSSL.Desc)
	}

	if len(kineSSL.SNIs) != 2 {
		t.Fatalf("Expected 2 SNIs, got %d", len(kineSSL.SNIs))
	}

	if kineSSL.SNIs[0] != "example.com" || kineSSL.SNIs[1] != "*.example.com" {
		t.Error("SNIs mismatch")
	}

	if !containsString(kineSSL.Cert, "test-cert") {
		t.Error("Certificate content mismatch")
	}

	if !containsString(kineSSL.Key, "test-key") {
		t.Error("Key content mismatch")
	}

	// Check labels
	if kineSSL.Labels["env"] != "prod" {
		t.Error("Labels mismatch")
	}
}

func TestTransferSSLSingleCertificateWithoutID(t *testing.T) {
	// Test SSL with single certificate without ID - should generate from name
	adcSSL := &adc.SSL{
		Metadata: adc.Metadata{
			ID:   "", // No ID provided
			Name: "test-ssl",
		},
		Certificates: []adc.Certificate{
			{
				Certificate: "cert-data",
				Key:         "key-data",
			},
		},
		Snis: []string{"example.com"},
	}

	kineSSLs, err := TransferSSL(adcSSL)
	if err != nil {
		t.Fatalf("TransferSSL failed: %v", err)
	}

	if len(kineSSLs) != 1 {
		t.Fatalf("Expected 1 SSL, got %d", len(kineSSLs))
	}

	kineSSL := kineSSLs[0]

	// Should generate ID from name
	expectedID := sha1Hash("test-ssl")
	if kineSSL.ID != expectedID {
		t.Errorf("Expected SSL ID to be generated as %s, got %s", expectedID, kineSSL.ID)
	}

	if kineSSL.Name != "test-ssl" {
		t.Errorf("Expected name 'test-ssl', got '%s'", kineSSL.Name)
	}
}

func TestTransferSSLMultipleCertificates(t *testing.T) {
	// Test SSL with multiple certificates
	adcSSL := &adc.SSL{
		Metadata: adc.Metadata{
			Name: "multi-cert-ssl",
			Desc: "SSL with multiple certificates",
		},
		Certificates: []adc.Certificate{
			{
				Certificate: "cert-1",
				Key:         "key-1",
			},
			{
				Certificate: "cert-2",
				Key:         "key-2",
			},
			{
				Certificate: "cert-3",
				Key:         "key-3",
			},
		},
		Snis: []string{"example.com", "example.org"},
	}

	kineSSLs, err := TransferSSL(adcSSL)
	if err != nil {
		t.Fatalf("TransferSSL failed: %v", err)
	}

	// Should create 3 Kine SSLs (one for each certificate)
	if len(kineSSLs) != 3 {
		t.Fatalf("Expected 3 SSLs, got %d", len(kineSSLs))
	}

	// Check each SSL
	for i, kineSSL := range kineSSLs {
		// ID should be generated with index
		expectedID := sha1Hash(fmt.Sprintf("multi-cert-ssl.%d", i))
		if kineSSL.ID != expectedID {
			t.Errorf("SSL %d: Expected ID %s, got %s", i, expectedID, kineSSL.ID)
		}

		// All should have the same name
		if kineSSL.Name != "multi-cert-ssl" {
			t.Errorf("SSL %d: Expected name 'multi-cert-ssl', got '%s'", i, kineSSL.Name)
		}

		// All should have the same SNIs
		if len(kineSSL.SNIs) != 2 {
			t.Errorf("SSL %d: Expected 2 SNIs, got %d", i, len(kineSSL.SNIs))
		}

		// Check certificate content
		expectedCert := fmt.Sprintf("cert-%d", i+1)
		if kineSSL.Cert != expectedCert {
			t.Errorf("SSL %d: Expected cert '%s', got '%s'", i, expectedCert, kineSSL.Cert)
		}

		// Check key content
		expectedKey := fmt.Sprintf("key-%d", i+1)
		if kineSSL.Key != expectedKey {
			t.Errorf("SSL %d: Expected key '%s', got '%s'", i, expectedKey, kineSSL.Key)
		}
	}
}

func TestTransferSSLNoCertificates(t *testing.T) {
	// Test SSL with no certificates - should fail
	adcSSL := &adc.SSL{
		Metadata: adc.Metadata{
			Name: "test-ssl",
		},
		Certificates: []adc.Certificate{},
		Snis:         []string{"example.com"},
	}

	_, err := TransferSSL(adcSSL)
	if err == nil {
		t.Error("Expected error for SSL with no certificates")
	}
}

func TestTransferSSLNoSnis(t *testing.T) {
	// Test SSL with no SNIs - should fail
	adcSSL := &adc.SSL{
		Metadata: adc.Metadata{
			Name: "test-ssl",
		},
		Certificates: []adc.Certificate{
			{
				Certificate: "cert",
				Key:         "key",
			},
		},
		Snis: []string{},
	}

	_, err := TransferSSL(adcSSL)
	if err == nil {
		t.Error("Expected error for SSL with no SNIs")
	}
}

func TestTransferSSLNil(t *testing.T) {
	// Test with nil SSL - should fail
	_, err := TransferSSL(nil)
	if err == nil {
		t.Error("Expected error for nil SSL")
	}
}

func TestTransferSSLClientCertificate(t *testing.T) {
	// Test with client certificate - should be ignored (return nil, nil)
	clientType := adc.Client
	adcSSL := &adc.SSL{
		Metadata: adc.Metadata{
			ID:   "client-cert-id",
			Name: "client-cert",
		},
		Type: &clientType,
		Certificates: []adc.Certificate{
			{
				Certificate: "-----BEGIN CERTIFICATE-----\nclient-cert\n-----END CERTIFICATE-----",
				Key:         "-----BEGIN PRIVATE KEY-----\nclient-key\n-----END PRIVATE KEY-----",
			},
		},
		Snis: []string{"client.example.com"},
	}

	kineSSLs, err := TransferSSL(adcSSL)
	if err != nil {
		t.Fatalf("TransferSSL should not fail for client certificate: %v", err)
	}

	if kineSSLs != nil {
		t.Errorf("Expected nil result for client certificate, got %d SSLs", len(kineSSLs))
	}
}

func TestTransferSSLServerCertificate(t *testing.T) {
	// Test with explicit server certificate - should work normally
	serverType := adc.Server
	adcSSL := &adc.SSL{
		Metadata: adc.Metadata{
			ID:   "server-cert-id",
			Name: "server-cert",
		},
		Type: &serverType,
		Certificates: []adc.Certificate{
			{
				Certificate: "-----BEGIN CERTIFICATE-----\nserver-cert\n-----END CERTIFICATE-----",
				Key:         "-----BEGIN PRIVATE KEY-----\nserver-key\n-----END PRIVATE KEY-----",
			},
		},
		Snis: []string{"server.example.com"},
	}

	kineSSLs, err := TransferSSL(adcSSL)
	if err != nil {
		t.Fatalf("TransferSSL failed: %v", err)
	}

	if len(kineSSLs) != 1 {
		t.Fatalf("Expected 1 SSL for server certificate, got %d", len(kineSSLs))
	}

	kineSSL := kineSSLs[0]
	if kineSSL.ID != "server-cert-id" {
		t.Errorf("Expected SSL ID 'server-cert-id', got '%s'", kineSSL.ID)
	}
}

func TestGenerateSSLID(t *testing.T) {
	// Test different scenarios for SSL ID generation

	// Single cert with ID
	ssl1 := &adc.SSL{
		Metadata: adc.Metadata{
			ID:   "custom-id",
			Name: "test",
		},
		Certificates: []adc.Certificate{{Certificate: "c", Key: "k"}},
	}
	id1 := generateSSLID(ssl1, 0)
	if id1 != "custom-id" {
		t.Errorf("Expected 'custom-id', got '%s'", id1)
	}

	// Single cert without ID
	ssl2 := &adc.SSL{
		Metadata: adc.Metadata{
			Name: "test-ssl",
		},
		Certificates: []adc.Certificate{{Certificate: "c", Key: "k"}},
	}
	id2 := generateSSLID(ssl2, 0)
	expectedID2 := sha1Hash("test-ssl")
	if id2 != expectedID2 {
		t.Errorf("Expected '%s', got '%s'", expectedID2, id2)
	}

	// Multiple certs
	ssl3 := &adc.SSL{
		Metadata: adc.Metadata{
			Name: "multi-ssl",
		},
		Certificates: []adc.Certificate{
			{Certificate: "c1", Key: "k1"},
			{Certificate: "c2", Key: "k2"},
		},
	}
	id3_0 := generateSSLID(ssl3, 0)
	id3_1 := generateSSLID(ssl3, 1)
	expectedID3_0 := sha1Hash("multi-ssl.0")
	expectedID3_1 := sha1Hash("multi-ssl.1")
	if id3_0 != expectedID3_0 {
		t.Errorf("Expected '%s', got '%s'", expectedID3_0, id3_0)
	}
	if id3_1 != expectedID3_1 {
		t.Errorf("Expected '%s', got '%s'", expectedID3_1, id3_1)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTransferGlobalRule(t *testing.T) {
	// Test GlobalRule with multiple plugins
	adcGlobalRule := adc.GlobalRule{
		"cors": map[string]any{
			"allow_origins": "**",
			"allow_methods": "GET,POST,PUT,DELETE",
		},
		"limit-req": map[string]any{
			"rate":  100,
			"burst": 50,
			"key":   "remote_addr",
		},
		"prometheus": map[string]any{
			"prefer_name": true,
		},
	}

	kineGlobalRules := TransferGlobalRule(adcGlobalRule)

	if len(kineGlobalRules) != 3 {
		t.Fatalf("Expected 3 global rules, got %d", len(kineGlobalRules))
	}

	// Check that each plugin has its own GlobalRule
	pluginNames := make(map[string]bool)
	for _, gr := range kineGlobalRules {
		// ID should be the plugin name
		if gr.ID == "" {
			t.Error("GlobalRule ID should not be empty")
		}

		pluginNames[gr.ID] = true

		// Each GlobalRule should have exactly one plugin
		if len(gr.Plugins) != 1 {
			t.Errorf("Expected 1 plugin in GlobalRule %s, got %d", gr.ID, len(gr.Plugins))
		}

		// The plugin name in Plugins should match the ID
		if _, exists := gr.Plugins[gr.ID]; !exists {
			t.Errorf("GlobalRule %s should contain plugin with name %s", gr.ID, gr.ID)
		}
	}

	// Verify all expected plugins are present
	expectedPlugins := []string{"cors", "limit-req", "prometheus"}
	for _, expectedPlugin := range expectedPlugins {
		if !pluginNames[expectedPlugin] {
			t.Errorf("Expected plugin %s not found in GlobalRules", expectedPlugin)
		}
	}
}

func TestTransferGlobalRuleSinglePlugin(t *testing.T) {
	// Test GlobalRule with single plugin
	adcGlobalRule := adc.GlobalRule{
		"ip-restriction": map[string]any{
			"whitelist": []string{"127.0.0.1", "192.168.1.0/24"},
		},
	}

	kineGlobalRules := TransferGlobalRule(adcGlobalRule)

	if len(kineGlobalRules) != 1 {
		t.Fatalf("Expected 1 global rule, got %d", len(kineGlobalRules))
	}

	gr := kineGlobalRules[0]

	if gr.ID != "ip-restriction" {
		t.Errorf("Expected ID 'ip-restriction', got '%s'", gr.ID)
	}

	if len(gr.Plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(gr.Plugins))
	}

	if _, exists := gr.Plugins["ip-restriction"]; !exists {
		t.Error("Plugin 'ip-restriction' not found in Plugins map")
	}

	// Verify plugin config
	pluginConfig, ok := gr.Plugins["ip-restriction"].(map[string]any)
	if !ok {
		t.Fatal("Plugin config is not a map")
	}

	whitelist, ok := pluginConfig["whitelist"].([]string)
	if !ok {
		t.Fatal("Whitelist is not a string slice")
	}

	if len(whitelist) != 2 {
		t.Errorf("Expected 2 whitelist entries, got %d", len(whitelist))
	}
}

func TestTransferGlobalRuleEmpty(t *testing.T) {
	// Test with empty GlobalRule
	adcGlobalRule := adc.GlobalRule{}

	kineGlobalRules := TransferGlobalRule(adcGlobalRule)

	if kineGlobalRules != nil {
		t.Errorf("Expected nil for empty GlobalRule, got %d rules", len(kineGlobalRules))
	}
}

func TestTransferGlobalRuleNil(t *testing.T) {
	// Test with nil GlobalRule
	var adcGlobalRule adc.GlobalRule = nil

	kineGlobalRules := TransferGlobalRule(adcGlobalRule)

	if kineGlobalRules != nil {
		t.Errorf("Expected nil for nil GlobalRule, got %d rules", len(kineGlobalRules))
	}
}

func TestTransferGlobalRuleComplexConfig(t *testing.T) {
	// Test GlobalRule with complex plugin configurations
	adcGlobalRule := adc.GlobalRule{
		"limit-count": map[string]any{
			"count":             100,
			"time_window":       60,
			"key":               "remote_addr",
			"rejected_code":     503,
			"rejected_msg":      "Too many requests",
			"policy":            "local",
			"allow_degradation": false,
		},
		"response-rewrite": map[string]any{
			"status_code": 200,
			"body":        "Modified response",
			"headers": map[string]string{
				"X-Custom-Header": "value",
			},
		},
	}

	kineGlobalRules := TransferGlobalRule(adcGlobalRule)

	if len(kineGlobalRules) != 2 {
		t.Fatalf("Expected 2 global rules, got %d", len(kineGlobalRules))
	}

	// Verify limit-count plugin
	var limitCountRule *GlobalRule
	var responseRewriteRule *GlobalRule

	for _, gr := range kineGlobalRules {
		switch gr.ID {
		case "limit-count":
			limitCountRule = gr
		case "response-rewrite":
			responseRewriteRule = gr
		}
	}

	if limitCountRule == nil {
		t.Fatal("limit-count GlobalRule not found")
	}

	if responseRewriteRule == nil {
		t.Fatal("response-rewrite GlobalRule not found")
	}

	// Verify limit-count config
	limitCountConfig, ok := limitCountRule.Plugins["limit-count"].(map[string]any)
	if !ok {
		t.Fatal("limit-count config is not a map")
	}

	if count, ok := limitCountConfig["count"].(int); !ok || count != 100 {
		t.Error("limit-count count mismatch")
	}

	// Verify response-rewrite config
	responseRewriteConfig, ok := responseRewriteRule.Plugins["response-rewrite"].(map[string]any)
	if !ok {
		t.Fatal("response-rewrite config is not a map")
	}

	if statusCode, ok := responseRewriteConfig["status_code"].(int); !ok || statusCode != 200 {
		t.Error("response-rewrite status_code mismatch")
	}
}

func TestTransferGlobalRulePluginOrder(t *testing.T) {
	// Test that all plugins are converted (order doesn't matter for map iteration)
	adcGlobalRule := adc.GlobalRule{
		"plugin-1": map[string]any{"config": 1},
		"plugin-2": map[string]any{"config": 2},
		"plugin-3": map[string]any{"config": 3},
		"plugin-4": map[string]any{"config": 4},
		"plugin-5": map[string]any{"config": 5},
	}

	kineGlobalRules := TransferGlobalRule(adcGlobalRule)

	if len(kineGlobalRules) != 5 {
		t.Fatalf("Expected 5 global rules, got %d", len(kineGlobalRules))
	}

	// Collect all IDs
	ids := make(map[string]bool)
	for _, gr := range kineGlobalRules {
		ids[gr.ID] = true

		// Verify each rule has exactly one plugin
		if len(gr.Plugins) != 1 {
			t.Errorf("GlobalRule %s should have exactly 1 plugin, got %d", gr.ID, len(gr.Plugins))
		}
	}

	// Verify all expected plugin IDs are present
	for i := 1; i <= 5; i++ {
		pluginName := fmt.Sprintf("plugin-%d", i)
		if !ids[pluginName] {
			t.Errorf("Expected plugin %s not found", pluginName)
		}
	}
}
