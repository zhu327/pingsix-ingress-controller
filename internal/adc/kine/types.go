package kine

import (
	"fmt"
	"regexp"

	"github.com/apache/apisix-ingress-controller/api/adc"
)

// NODE_KEY_REGEX for validating node keys
var NODE_KEY_REGEX = regexp.MustCompile(`^[a-zA-Z0-9\.\-_:]+$`)

// Method represents HTTP methods
type Method string

const (
	MethodGET     Method = "GET"
	MethodPOST    Method = "POST"
	MethodPUT     Method = "PUT"
	MethodDELETE  Method = "DELETE"
	MethodPATCH   Method = "PATCH"
	MethodHEAD    Method = "HEAD"
	MethodOPTIONS Method = "OPTIONS"
)

// SelectionType represents upstream selection algorithms
type SelectionType string

const (
	SelectionTypeRoundRobin SelectionType = "roundrobin"
	SelectionTypeRandom     SelectionType = "random"
	SelectionTypeFnv        SelectionType = "fnv"
	SelectionTypeKetama     SelectionType = "ketama"
)

// ActiveCheckType represents active health check types
type ActiveCheckType string

const (
	ActiveCheckTypeTCP   ActiveCheckType = "tcp"
	ActiveCheckTypeHTTP  ActiveCheckType = "http"
	ActiveCheckTypeHTTPS ActiveCheckType = "https"
)

// UpstreamHashOn represents upstream hash on options
type UpstreamHashOn string

const (
	UpstreamHashOnVars   UpstreamHashOn = "vars"
	UpstreamHashOnHead   UpstreamHashOn = "head"
	UpstreamHashOnCookie UpstreamHashOn = "cookie"
)

// UpstreamScheme represents upstream schemes
type UpstreamScheme string

const (
	UpstreamSchemeHTTP  UpstreamScheme = "http"
	UpstreamSchemeHTTPS UpstreamScheme = "https"
	UpstreamSchemeGRPC  UpstreamScheme = "grpc"
	UpstreamSchemeGRPCS UpstreamScheme = "grpcs"
)

// UpstreamPassHost represents upstream pass host options
type UpstreamPassHost string

const (
	UpstreamPassHostPass    UpstreamPassHost = "pass"
	UpstreamPassHostRewrite UpstreamPassHost = "rewrite"
	UpstreamPassHostNode    UpstreamPassHost = "node"
)

// Timeout represents timeout configuration
type Timeout struct {
	Connect int `json:"connect,omitempty"`
	Send    int `json:"send,omitempty"`
	Read    int `json:"read,omitempty"`
}

// Route represents an APISIX route
type Route struct {
	adc.Metadata `json:",inline"`

	URI        *string        `json:"uri,omitempty"`
	URIs       []string       `json:"uris,omitempty"`
	Methods    []Method       `json:"methods,omitempty"`
	Host       *string        `json:"host,omitempty"`
	Hosts      []string       `json:"hosts,omitempty"`
	Priority   uint32         `json:"priority,omitempty"`
	Plugins    map[string]any `json:"plugins,omitempty"`
	Upstream   *Upstream      `json:"upstream,omitempty"`
	UpstreamID *string        `json:"upstream_id,omitempty"`
	ServiceID  *string        `json:"service_id,omitempty"`
	Timeout    *Timeout       `json:"timeout,omitempty"`
}

// Validate validates the Route
func (r *Route) Validate() error {
	if r.URI == nil && len(r.URIs) == 0 {
		return fmt.Errorf("uri or uris is required")
	}

	if r.UpstreamID == nil && r.ServiceID == nil && r.Upstream == nil {
		return fmt.Errorf("upstream, upstream_id, or service_id is required")
	}

	if r.Upstream != nil {
		if err := r.Upstream.Validate(); err != nil {
			return err
		}
	}

	if r.Timeout != nil {
		if err := r.Timeout.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetHosts returns the hosts for the route
func (r *Route) GetHosts() []string {
	if r.Host != nil {
		return []string{*r.Host}
	}
	return r.Hosts
}

// GetURIs returns the URIs for the route
func (r *Route) GetURIs() []string {
	if r.URI != nil {
		return []string{*r.URI}
	}
	return r.URIs
}

// GetPriority returns the priority with default value
func (r *Route) GetPriority() uint32 {
	if r.Priority == 0 {
		return 0 // default priority
	}
	return r.Priority
}

// Upstream represents an APISIX upstream
type Upstream struct {
	adc.Metadata `json:",inline"`

	Retries      *uint32           `json:"retries,omitempty"`
	RetryTimeout *uint64           `json:"retry_timeout,omitempty"`
	Timeout      *Timeout          `json:"timeout,omitempty"`
	Nodes        map[string]uint32 `json:"nodes"`
	Type         SelectionType     `json:"type,omitempty"`
	Checks       *HealthCheck      `json:"checks,omitempty"`
	HashOn       UpstreamHashOn    `json:"hash_on,omitempty"`
	Key          string            `json:"key,omitempty"`
	Scheme       UpstreamScheme    `json:"scheme,omitempty"`
	PassHost     UpstreamPassHost  `json:"pass_host,omitempty"`
	UpstreamHost *string           `json:"upstream_host,omitempty"`
}

// Validate validates the Upstream
func (u *Upstream) Validate() error {
	if len(u.Nodes) == 0 {
		return fmt.Errorf("nodes cannot be empty")
	}

	// Validate node keys
	for key := range u.Nodes {
		if !NODE_KEY_REGEX.MatchString(key) {
			return fmt.Errorf("invalid node key: %s", key)
		}
	}

	if u.PassHost == UpstreamPassHostRewrite && u.UpstreamHost == nil {
		return fmt.Errorf("upstream_host is required when pass_host is rewrite")
	}

	if u.Checks != nil {
		if err := u.Checks.Validate(); err != nil {
			return err
		}
	}

	if u.Timeout != nil {
		if err := u.Timeout.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetKey returns the key with default value
func (u *Upstream) GetKey() string {
	if u.Key == "" {
		return "uri"
	}
	return u.Key
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Active *ActiveCheck `json:"active,omitempty"`
}

// Validate validates the HealthCheck
func (h *HealthCheck) Validate() error {
	if h.Active != nil {
		return h.Active.Validate()
	}
	return nil
}

// ActiveCheck represents active health check configuration
type ActiveCheck struct {
	Type                   ActiveCheckType `json:"type,omitempty"`
	Timeout                uint32          `json:"timeout,omitempty"`
	HTTPPath               string          `json:"http_path,omitempty"`
	Host                   *string         `json:"host,omitempty"`
	Port                   *uint32         `json:"port,omitempty"`
	HTTPSVerifyCertificate bool            `json:"https_verify_certificate,omitempty"`
	ReqHeaders             []string        `json:"req_headers,omitempty"`
	Healthy                *Health         `json:"healthy,omitempty"`
	Unhealthy              *Unhealthy      `json:"unhealthy,omitempty"`
}

// Validate validates the ActiveCheck
func (a *ActiveCheck) Validate() error {
	if a.Unhealthy != nil {
		return a.Unhealthy.Validate()
	}
	return nil
}

// GetTimeout returns the timeout with default value
func (a *ActiveCheck) GetTimeout() uint32 {
	if a.Timeout == 0 {
		return 1
	}
	return a.Timeout
}

// GetHTTPPath returns the HTTP path with default value
func (a *ActiveCheck) GetHTTPPath() string {
	if a.HTTPPath == "" {
		return "/"
	}
	return a.HTTPPath
}

// GetHTTPSVerifyCertificate returns the HTTPS verify certificate with default value
func (a *ActiveCheck) GetHTTPSVerifyCertificate() bool {
	return a.HTTPSVerifyCertificate // default is true in the original code
}

// Health represents healthy check configuration
type Health struct {
	Interval     uint32   `json:"interval,omitempty"`
	HTTPStatuses []uint32 `json:"http_statuses,omitempty"`
	Successes    uint32   `json:"successes,omitempty"`
}

// GetInterval returns the interval with default value
func (h *Health) GetInterval() uint32 {
	if h.Interval == 0 {
		return 1
	}
	return h.Interval
}

// GetHTTPStatuses returns the HTTP statuses with default values
func (h *Health) GetHTTPStatuses() []uint32 {
	if len(h.HTTPStatuses) == 0 {
		return []uint32{200, 302}
	}
	return h.HTTPStatuses
}

// GetSuccesses returns the successes with default value
func (h *Health) GetSuccesses() uint32 {
	if h.Successes == 0 {
		return 2
	}
	return h.Successes
}

// Unhealthy represents unhealthy check configuration
type Unhealthy struct {
	HTTPFailures uint32 `json:"http_failures,omitempty"`
	TCPFailures  uint32 `json:"tcp_failures,omitempty"`
}

// Validate validates the Unhealthy
func (u *Unhealthy) Validate() error {
	return nil
}

// GetHTTPFailures returns the HTTP failures with default value
func (u *Unhealthy) GetHTTPFailures() uint32 {
	if u.HTTPFailures == 0 {
		return 5
	}
	return u.HTTPFailures
}

// GetTCPFailures returns the TCP failures with default value
func (u *Unhealthy) GetTCPFailures() uint32 {
	if u.TCPFailures == 0 {
		return 2
	}
	return u.TCPFailures
}

// Service represents an APISIX service
type Service struct {
	adc.Metadata `json:",inline"`

	Plugins    map[string]any `json:"plugins,omitempty"`
	Upstream   *Upstream      `json:"upstream,omitempty"`
	UpstreamID *string        `json:"upstream_id,omitempty"`
	Hosts      []string       `json:"hosts,omitempty"`
}

// Validate validates the Service
func (s *Service) Validate() error {
	if s.UpstreamID == nil && s.Upstream == nil {
		return fmt.Errorf("upstream or upstream_id is required")
	}

	if s.Upstream != nil {
		return s.Upstream.Validate()
	}

	return nil
}

// GlobalRule represents an APISIX global rule
type GlobalRule struct {
	ID      string         `json:"id,omitempty"`
	Plugins map[string]any `json:"plugins,omitempty"`
}

// Validate validates the GlobalRule
func (g *GlobalRule) Validate() error {
	return nil
}

// SSL represents an APISIX SSL certificate
type SSL struct {
	adc.Metadata `json:",inline"`

	Cert string   `json:"cert"`
	Key  string   `json:"key"`
	SNIs []string `json:"snis"`
}

// Validate validates the SSL
func (s *SSL) Validate() error {
	if len(s.SNIs) == 0 {
		return fmt.Errorf("snis cannot be empty")
	}
	if s.Cert == "" {
		return fmt.Errorf("cert is required")
	}
	if s.Key == "" {
		return fmt.Errorf("key is required")
	}
	return nil
}

// Validate validates the Timeout
func (t *Timeout) Validate() error {
	return nil
}
