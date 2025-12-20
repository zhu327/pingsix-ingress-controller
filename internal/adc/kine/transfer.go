package kine

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/apache/apisix-ingress-controller/api/adc"
)

// TransferService converts an ADC Service to Kine Service and Routes
func TransferService(adcSvc *adc.Service) (*Service, []*Route, error) {
	if adcSvc == nil {
		return nil, nil, fmt.Errorf("adc service is nil")
	}

	// Ignore Upstreams, only use Upstream
	if adcSvc.Upstream == nil {
		return nil, nil, fmt.Errorf("adc service upstream is nil")
	}

	// Convert ADC Service to Kine Service
	kineSvc := &Service{
		Metadata: adc.Metadata{
			ID:     generateServiceID(adcSvc),
			Name:   adcSvc.Name,
			Desc:   adcSvc.Desc,
			Labels: copyLabels(adcSvc.Labels),
		},
		Plugins:  convertPlugins(adcSvc.Plugins),
		Upstream: convertUpstream(adcSvc.Upstream),
		Hosts:    copyStringSlice(adcSvc.Hosts),
	}

	// Convert ADC Routes to Kine Routes
	kineRoutes := make([]*Route, 0, len(adcSvc.Routes))
	for _, adcRoute := range adcSvc.Routes {
		kineRoute, err := convertRoute(adcRoute, adcSvc)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert route: %w", err)
		}
		kineRoutes = append(kineRoutes, kineRoute)
	}

	return kineSvc, kineRoutes, nil
}

// generateServiceID generates service ID from name using SHA1
func generateServiceID(adcSvc *adc.Service) string {
	if adcSvc.ID != "" {
		return adcSvc.ID
	}
	return sha1Hash(adcSvc.Name)
}

// generateRouteID generates route ID from service name and route name using SHA1
func generateRouteID(adcRoute *adc.Route, adcSvc *adc.Service) string {
	if adcRoute.ID != "" {
		return adcRoute.ID
	}
	return sha1Hash(adcSvc.Name + "." + adcRoute.Name)
}

// sha1Hash generates SHA1 hash of the input string
func sha1Hash(input string) string {
	hash := sha1.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}

// convertRoute converts an ADC Route to Kine Route
func convertRoute(adcRoute *adc.Route, adcSvc *adc.Service) (*Route, error) {
	if adcRoute == nil {
		return nil, fmt.Errorf("adc route is nil")
	}

	kineRoute := &Route{
		Metadata: adc.Metadata{
			ID:     generateRouteID(adcRoute, adcSvc),
			Name:   adcRoute.Name,
			Desc:   adcRoute.Desc,
			Labels: copyLabels(adcRoute.Labels),
		},
		URIs:    copyStringSlice(adcRoute.Uris),
		Methods: convertMethods(adcRoute.Methods),
		Hosts:   copyStringSlice(adcRoute.Hosts),
		Plugins: convertPlugins(adcRoute.Plugins),
		Timeout: convertTimeout(adcRoute.Timeout),
	}

	// Set ServiceID to reference the parent service
	serviceID := generateServiceID(adcSvc)
	kineRoute.ServiceID = &serviceID

	// Convert priority
	if adcRoute.Priority != nil {
		// ADC uses int64, Kine uses uint32
		kineRoute.Priority = uint32(*adcRoute.Priority)
	}

	return kineRoute, nil
}

// convertUpstream converts ADC Upstream to Kine Upstream
func convertUpstream(adcUpstream *adc.Upstream) *Upstream {
	if adcUpstream == nil {
		return nil
	}

	// Generate upstream ID if not provided
	upstreamID := adcUpstream.ID
	if upstreamID == "" && adcUpstream.Name != "" {
		upstreamID = sha1Hash(adcUpstream.Name)
	}

	kineUpstream := &Upstream{
		Metadata: adc.Metadata{
			ID:     upstreamID,
			Name:   adcUpstream.Name,
			Desc:   adcUpstream.Desc,
			Labels: copyLabels(adcUpstream.Labels),
		},
		Nodes:    convertNodes(adcUpstream.Nodes),
		Type:     convertUpstreamType(adcUpstream.Type),
		HashOn:   convertHashOn(adcUpstream.HashOn),
		Key:      adcUpstream.Key,
		Scheme:   convertScheme(adcUpstream.Scheme),
		PassHost: convertPassHost(adcUpstream.PassHost),
		Timeout:  convertTimeout(adcUpstream.Timeout),
		Checks:   convertHealthCheck(adcUpstream.Checks),
	}

	// Convert retries
	if adcUpstream.Retries != nil {
		retries := uint32(*adcUpstream.Retries)
		kineUpstream.Retries = &retries
	}

	// Convert retry_timeout
	if adcUpstream.RetryTimeout != nil {
		retryTimeout := uint64(*adcUpstream.RetryTimeout)
		kineUpstream.RetryTimeout = &retryTimeout
	}

	// Convert upstream_host
	if adcUpstream.UpstreamHost != "" {
		kineUpstream.UpstreamHost = &adcUpstream.UpstreamHost
	}

	return kineUpstream
}

// convertNodes converts ADC UpstreamNodes to Kine nodes map
func convertNodes(adcNodes adc.UpstreamNodes) map[string]uint32 {
	nodes := make(map[string]uint32)
	for _, node := range adcNodes {
		key := node.Host + ":" + strconv.Itoa(node.Port)
		nodes[key] = uint32(node.Weight)
	}
	return nodes
}

// convertUpstreamType converts ADC UpstreamType to Kine SelectionType
func convertUpstreamType(adcType adc.UpstreamType) SelectionType {
	switch adcType {
	case adc.Roundrobin:
		return SelectionTypeRoundRobin
	case adc.Random:
		return SelectionTypeRandom
	case adc.Chash:
		return SelectionTypeFnv
	case adc.Ketama:
		return SelectionTypeKetama
	case adc.LeastConn:
		return SelectionTypeRoundRobin // fallback
	case adc.Ewma:
		return SelectionTypeRoundRobin // fallback
	default:
		return SelectionTypeRoundRobin
	}
}

// convertHashOn converts ADC hash_on to Kine UpstreamHashOn
func convertHashOn(hashOn string) UpstreamHashOn {
	switch hashOn {
	case "vars":
		return UpstreamHashOnVars
	case "header":
		return UpstreamHashOnHead
	case "cookie":
		return UpstreamHashOnCookie
	default:
		return UpstreamHashOnVars
	}
}

// convertScheme converts ADC scheme to Kine UpstreamScheme
func convertScheme(scheme string) UpstreamScheme {
	switch scheme {
	case "http":
		return UpstreamSchemeHTTP
	case "https":
		return UpstreamSchemeHTTPS
	case "grpc":
		return UpstreamSchemeGRPC
	case "grpcs":
		return UpstreamSchemeGRPCS
	default:
		return UpstreamSchemeHTTP
	}
}

// convertPassHost converts ADC pass_host to Kine UpstreamPassHost
func convertPassHost(passHost string) UpstreamPassHost {
	switch passHost {
	case "pass":
		return UpstreamPassHostPass
	case "rewrite":
		return UpstreamPassHostRewrite
	default:
		return UpstreamPassHostPass
	}
}

// convertTimeout converts ADC Timeout to Kine Timeout
func convertTimeout(adcTimeout *adc.Timeout) *Timeout {
	if adcTimeout == nil {
		return nil
	}
	return &Timeout{
		Connect: adcTimeout.Connect,
		Send:    adcTimeout.Send,
		Read:    adcTimeout.Read,
	}
}

// convertHealthCheck converts ADC health check to Kine health check
func convertHealthCheck(adcCheck *adc.UpstreamHealthCheck) *HealthCheck {
	if adcCheck == nil || adcCheck.Active == nil {
		return nil
	}

	kineCheck := &HealthCheck{
		Active: &ActiveCheck{
			Type:       convertActiveCheckType(adcCheck.Active.Type),
			Timeout:    uint32(adcCheck.Active.Timeout),
			HTTPPath:   adcCheck.Active.HTTPPath,
			ReqHeaders: copyStringSlice(adcCheck.Active.HTTPRequestHeaders),
		},
	}

	// Convert host
	if adcCheck.Active.Host != "" {
		kineCheck.Active.Host = &adcCheck.Active.Host
	}

	// Convert port
	if adcCheck.Active.Port != 0 {
		port := uint32(adcCheck.Active.Port)
		kineCheck.Active.Port = &port
	}

	// Convert HTTPS verify certificate
	kineCheck.Active.HTTPSVerifyCertificate = adcCheck.Active.HTTPSVerifyCert

	// Convert healthy
	kineCheck.Active.Healthy = &Health{
		Interval:     uint32(adcCheck.Active.Healthy.Interval),
		HTTPStatuses: convertIntSliceToUint32(adcCheck.Active.Healthy.HTTPStatuses),
		Successes:    uint32(adcCheck.Active.Healthy.Successes),
	}

	// Convert unhealthy
	kineCheck.Active.Unhealthy = &Unhealthy{
		HTTPFailures: uint32(adcCheck.Active.Unhealthy.HTTPFailures),
		TCPFailures:  uint32(adcCheck.Active.Unhealthy.TCPFailures),
	}

	return kineCheck
}

// convertActiveCheckType converts ADC active check type to Kine ActiveCheckType
func convertActiveCheckType(checkType string) ActiveCheckType {
	switch checkType {
	case "tcp":
		return ActiveCheckTypeTCP
	case "http":
		return ActiveCheckTypeHTTP
	case "https":
		return ActiveCheckTypeHTTPS
	default:
		return ActiveCheckTypeHTTP
	}
}

// convertMethods converts ADC methods to Kine methods
func convertMethods(adcMethods []string) []Method {
	methods := make([]Method, 0, len(adcMethods))
	for _, m := range adcMethods {
		methods = append(methods, Method(m))
	}
	return methods
}

// convertPlugins converts ADC plugins to Kine plugins
func convertPlugins(adcPlugins adc.Plugins) map[string]any {
	if adcPlugins == nil {
		return nil
	}
	plugins := make(map[string]any, len(adcPlugins))
	for k, v := range adcPlugins {
		plugins[k] = v
	}
	return plugins
}

// copyLabels creates a copy of labels map
func copyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}
	copied := make(map[string]string, len(labels))
	for k, v := range labels {
		copied[k] = v
	}
	return copied
}

// copyStringSlice creates a copy of string slice
func copyStringSlice(slice []string) []string {
	if slice == nil {
		return nil
	}
	copied := make([]string, len(slice))
	copy(copied, slice)
	return copied
}

// convertIntSliceToUint32 converts []int to []uint32
func convertIntSliceToUint32(intSlice []int) []uint32 {
	uint32Slice := make([]uint32, len(intSlice))
	for i, v := range intSlice {
		uint32Slice[i] = uint32(v)
	}
	return uint32Slice
}

// TransferSSL converts an ADC SSL to Kine SSL(s)
// Since ADC SSL supports multiple certificates and Kine SSL supports only one,
// this function returns multiple Kine SSLs if there are multiple certificates.
// Note: Kine does not support client certificates, so client-type SSLs are ignored.
func TransferSSL(adcSSL *adc.SSL) ([]*SSL, error) {
	if adcSSL == nil {
		return nil, fmt.Errorf("adc ssl is nil")
	}

	// Skip client certificates - Kine only supports server certificates
	if adcSSL.Type != nil && *adcSSL.Type == adc.Client {
		return nil, nil
	}

	if len(adcSSL.Certificates) == 0 {
		return nil, fmt.Errorf("adc ssl has no certificates")
	}

	if len(adcSSL.Snis) == 0 {
		return nil, fmt.Errorf("adc ssl has no snis")
	}

	kineSSLs := make([]*SSL, 0, len(adcSSL.Certificates))

	// For each certificate in ADC SSL, create a Kine SSL
	// All certificates share the same SNIs
	for i, cert := range adcSSL.Certificates {
		sslID := generateSSLID(adcSSL, i)

		kineSSL := &SSL{
			Metadata: adc.Metadata{
				ID:     sslID,
				Name:   adcSSL.Name,
				Desc:   adcSSL.Desc,
				Labels: copyLabels(adcSSL.Labels),
			},
			Cert: cert.Certificate,
			Key:  cert.Key,
			SNIs: copyStringSlice(adcSSL.Snis),
		}

		kineSSLs = append(kineSSLs, kineSSL)
	}

	return kineSSLs, nil
}

// generateSSLID generates SSL ID
// If there's only one certificate and ID is provided, use it
// If there's only one certificate and no ID, use sha1(name)
// If there are multiple certificates, use sha1(name.index)
func generateSSLID(adcSSL *adc.SSL, index int) string {
	// If only one certificate and ID is provided, use it
	if len(adcSSL.Certificates) == 1 && adcSSL.ID != "" {
		return adcSSL.ID
	}

	// If only one certificate and no ID, generate from name
	if len(adcSSL.Certificates) == 1 && adcSSL.Name != "" {
		return sha1Hash(adcSSL.Name)
	}

	// Multiple certificates - append index to name
	if adcSSL.Name != "" {
		return sha1Hash(fmt.Sprintf("%s.%d", adcSSL.Name, index))
	}

	// Fallback: use ID with index
	if adcSSL.ID != "" {
		return fmt.Sprintf("%s-%d", adcSSL.ID, index)
	}

	// Last resort: generate from index
	return sha1Hash(fmt.Sprintf("ssl-%d", index))
}

// TransferGlobalRule converts an ADC GlobalRule to Kine GlobalRules
// Each plugin in the ADC GlobalRule becomes a separate Kine GlobalRule
// The plugin name is used as the ID
func TransferGlobalRule(adcGlobalRule adc.GlobalRule) []*GlobalRule {
	if len(adcGlobalRule) == 0 {
		return nil
	}

	kineGlobalRules := make([]*GlobalRule, 0, len(adcGlobalRule))

	// Each plugin becomes a separate GlobalRule
	for pluginName, pluginConfig := range adcGlobalRule {
		kineGlobalRule := &GlobalRule{
			ID: pluginName, // Use plugin name as ID
			Plugins: map[string]any{
				pluginName: pluginConfig,
			},
		}
		kineGlobalRules = append(kineGlobalRules, kineGlobalRule)
	}

	return kineGlobalRules
}
