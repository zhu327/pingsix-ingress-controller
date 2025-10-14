package kine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-memdb"

	"github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
)

const (
	KineLabelIndex = "label"
)

var (
	// ErrStillInUse means an object is still in use.
	ErrStillInUse = errors.New("still in use")
	// ErrNotFound is returned when the requested item is not found.
	ErrNotFound = memdb.ErrNotFound
)

// =============================================================================
// Schema Definition
// =============================================================================

var _schema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		"route": {
			Name: "route",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "ID"},
				},
				"label": {
					Name:         "label",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &KineLabelIndexer,
				},
			},
		},
		"service": {
			Name: "service",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "ID"},
				},
				"label": {
					Name:         "label",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &KineLabelIndexer,
				},
			},
		},
		"upstream": {
			Name: "upstream",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "ID"},
				},
				"label": {
					Name:         "label",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &KineLabelIndexer,
				},
			},
		},
		"ssl": {
			Name: "ssl",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "ID"},
				},
				"label": {
					Name:         "label",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &KineLabelIndexer,
				},
			},
		},
		"global_rule": {
			Name: "global_rule",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "ID"},
				},
			},
		},
	},
}

// =============================================================================
// Label Indexer
// =============================================================================

var KineLabelIndexer = LabelIndexer{
	LabelKeys: []string{label.LabelKind, label.LabelNamespace, label.LabelName},
	GetLabels: func(obj any) map[string]string {
		switch t := obj.(type) {
		case *Route:
			return t.Labels
		case *Service:
			return t.Labels
		case *Upstream:
			return t.Labels
		case *SSL:
			return t.Labels
		default:
			return nil
		}
	},
}

type LabelIndexer struct {
	LabelKeys []string
	GetLabels func(obj any) map[string]string
}

// ref: https://pkg.go.dev/github.com/hashicorp/go-memdb#Txn.Get
// by adding suffixes to avoid prefix matching
func (li *LabelIndexer) genKey(labelValues []string) []byte {
	return []byte(strings.Join(labelValues, "/") + "\x00")
}

func (li *LabelIndexer) FromObject(obj any) (bool, []byte, error) {
	labels := li.GetLabels(obj)
	if labels == nil {
		return false, nil, nil
	}

	var labelValues []string
	for _, key := range li.LabelKeys {
		if value, exists := labels[key]; exists {
			labelValues = append(labelValues, value)
		}
	}

	if len(labelValues) == 0 {
		return false, nil, nil
	}

	return true, li.genKey(labelValues), nil
}

func (li *LabelIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != len(li.LabelKeys) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(li.LabelKeys), len(args))
	}

	labelValues := make([]string, 0, len(args))
	for _, arg := range args {
		value, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("argument is not a string")
		}
		labelValues = append(labelValues, value)
	}

	return li.genKey(labelValues), nil
}

// =============================================================================
// Cache Interface
// =============================================================================

// Cache interface for Kine types
type Cache interface {
	// Insert adds or updates an object to cache
	Insert(obj any) error
	// Delete removes an object from cache
	Delete(obj any) error

	// InsertRoute adds or updates route to cache
	InsertRoute(*Route) error
	// InsertService adds or updates service to cache
	InsertService(*Service) error
	// InsertUpstream adds or updates upstream to cache
	InsertUpstream(*Upstream) error
	// InsertSSL adds or updates SSL to cache
	InsertSSL(*SSL) error
	// InsertGlobalRule adds or updates global rule to cache
	InsertGlobalRule(*GlobalRule) error

	// GetRoute finds the route from cache according to the primary index (id)
	GetRoute(string) (*Route, error)
	// GetService finds the service from cache according to the primary index (id)
	GetService(string) (*Service, error)
	// GetUpstream finds the upstream from cache according to the primary index (id)
	GetUpstream(string) (*Upstream, error)
	// GetSSL finds the SSL from cache according to the primary index (id)
	GetSSL(string) (*SSL, error)
	// GetGlobalRule finds the global rule from cache according to the primary index (id)
	GetGlobalRule(string) (*GlobalRule, error)

	// DeleteRoute deletes the specified route in cache
	DeleteRoute(*Route) error
	// DeleteService deletes the specified service in cache
	DeleteService(*Service) error
	// DeleteUpstream deletes the specified upstream in cache
	DeleteUpstream(*Upstream) error
	// DeleteSSL deletes the specified SSL in cache
	DeleteSSL(*SSL) error
	// DeleteGlobalRule deletes the specified global rule in cache
	DeleteGlobalRule(*GlobalRule) error

	// ListRoutes lists all route objects in cache
	ListRoutes(...ListOption) ([]*Route, error)
	// ListServices lists all service objects in cache
	ListServices(...ListOption) ([]*Service, error)
	// ListUpstreams lists all upstream objects in cache
	ListUpstreams(...ListOption) ([]*Upstream, error)
	// ListSSL lists all SSL objects in cache
	ListSSL(...ListOption) ([]*SSL, error)
	// ListGlobalRules lists all global rule objects in cache
	ListGlobalRules(...ListOption) ([]*GlobalRule, error)
}

// ListOption interface for list options
type ListOption interface {
	ApplyToList(*ListOptions)
}

// ListOptions contains filtering options for list operations
type ListOptions struct {
	KindLabelSelector *KindLabelSelector
}

func (o *ListOptions) ApplyToList(lo *ListOptions) {
	if o.KindLabelSelector != nil {
		lo.KindLabelSelector = o.KindLabelSelector
	}
}

func (o *ListOptions) ApplyOptions(opts []ListOption) *ListOptions {
	for _, opt := range opts {
		opt.ApplyToList(o)
	}
	return o
}

// KindLabelSelector is used to filter objects by label
type KindLabelSelector struct {
	Kind      string
	Name      string
	Namespace string
}

func (o *KindLabelSelector) ApplyToList(opts *ListOptions) {
	opts.KindLabelSelector = o
}

// =============================================================================
// Cache Implementation
// =============================================================================

type dbCache struct {
	db *memdb.MemDB
}

// NewMemDBCache creates a Cache object backed with a memory DB
func NewMemDBCache() (Cache, error) {
	db, err := memdb.NewMemDB(_schema)
	if err != nil {
		return nil, err
	}
	return &dbCache{
		db: db,
	}, nil
}

func (c *dbCache) Insert(obj any) error {
	switch t := obj.(type) {
	case *Route:
		return c.InsertRoute(t)
	case *Service:
		return c.InsertService(t)
	case *Upstream:
		return c.InsertUpstream(t)
	case *SSL:
		return c.InsertSSL(t)
	case *GlobalRule:
		return c.InsertGlobalRule(t)
	default:
		return errors.New("unsupported type")
	}
}

func (c *dbCache) Delete(obj any) error {
	switch t := obj.(type) {
	case *Route:
		return c.DeleteRoute(t)
	case *Service:
		return c.DeleteService(t)
	case *Upstream:
		return c.DeleteUpstream(t)
	case *SSL:
		return c.DeleteSSL(t)
	case *GlobalRule:
		return c.DeleteGlobalRule(t)
	default:
		return errors.New("unsupported type")
	}
}

// Insert methods
func (c *dbCache) InsertRoute(r *Route) error {
	return c.insert("route", r.DeepCopy())
}

func (c *dbCache) InsertService(s *Service) error {
	return c.insert("service", s.DeepCopy())
}

func (c *dbCache) InsertUpstream(u *Upstream) error {
	return c.insert("upstream", u.DeepCopy())
}

func (c *dbCache) InsertSSL(ssl *SSL) error {
	return c.insert("ssl", ssl.DeepCopy())
}

func (c *dbCache) InsertGlobalRule(gr *GlobalRule) error {
	return c.insert("global_rule", gr.DeepCopy())
}

func (c *dbCache) insert(table string, obj any) error {
	txn := c.db.Txn(true)
	defer txn.Abort()
	if err := txn.Insert(table, obj); err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// Get methods
func (c *dbCache) GetRoute(id string) (*Route, error) {
	obj, err := c.get("route", id)
	if err != nil {
		return nil, err
	}
	return obj.(*Route).DeepCopy(), nil
}

func (c *dbCache) GetService(id string) (*Service, error) {
	obj, err := c.get("service", id)
	if err != nil {
		return nil, err
	}
	return obj.(*Service).DeepCopy(), nil
}

func (c *dbCache) GetUpstream(id string) (*Upstream, error) {
	obj, err := c.get("upstream", id)
	if err != nil {
		return nil, err
	}
	return obj.(*Upstream).DeepCopy(), nil
}

func (c *dbCache) GetSSL(id string) (*SSL, error) {
	obj, err := c.get("ssl", id)
	if err != nil {
		return nil, err
	}
	return obj.(*SSL).DeepCopy(), nil
}

func (c *dbCache) GetGlobalRule(id string) (*GlobalRule, error) {
	obj, err := c.get("global_rule", id)
	if err != nil {
		return nil, err
	}
	return obj.(*GlobalRule).DeepCopy(), nil
}

func (c *dbCache) get(table, id string) (any, error) {
	txn := c.db.Txn(false)
	defer txn.Abort()
	obj, err := txn.First(table, "id", id)
	if err != nil {
		if err == memdb.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if obj == nil {
		return nil, ErrNotFound
	}
	return obj, nil
}

// List methods
func (c *dbCache) ListRoutes(opts ...ListOption) ([]*Route, error) {
	raws, err := c.list("route", opts...)
	if err != nil {
		return nil, err
	}
	routes := make([]*Route, 0, len(raws))
	for _, raw := range raws {
		routes = append(routes, raw.(*Route).DeepCopy())
	}
	return routes, nil
}

func (c *dbCache) ListServices(opts ...ListOption) ([]*Service, error) {
	raws, err := c.list("service", opts...)
	if err != nil {
		return nil, err
	}
	services := make([]*Service, 0, len(raws))
	for _, raw := range raws {
		services = append(services, raw.(*Service).DeepCopy())
	}
	return services, nil
}

func (c *dbCache) ListUpstreams(opts ...ListOption) ([]*Upstream, error) {
	raws, err := c.list("upstream", opts...)
	if err != nil {
		return nil, err
	}
	upstreams := make([]*Upstream, 0, len(raws))
	for _, raw := range raws {
		upstreams = append(upstreams, raw.(*Upstream).DeepCopy())
	}
	return upstreams, nil
}

func (c *dbCache) ListSSL(opts ...ListOption) ([]*SSL, error) {
	raws, err := c.list("ssl", opts...)
	if err != nil {
		return nil, err
	}
	ssls := make([]*SSL, 0, len(raws))
	for _, raw := range raws {
		ssls = append(ssls, raw.(*SSL).DeepCopy())
	}
	return ssls, nil
}

func (c *dbCache) ListGlobalRules(opts ...ListOption) ([]*GlobalRule, error) {
	raws, err := c.list("global_rule", opts...)
	if err != nil {
		return nil, err
	}
	globalRules := make([]*GlobalRule, 0, len(raws))
	for _, raw := range raws {
		globalRules = append(globalRules, raw.(*GlobalRule).DeepCopy())
	}
	return globalRules, nil
}

func (c *dbCache) list(table string, opts ...ListOption) ([]any, error) {
	txn := c.db.Txn(false)
	defer txn.Abort()
	listOpts := &ListOptions{}
	listOpts.ApplyOptions(opts)
	index := "id"
	var args []any
	if listOpts.KindLabelSelector != nil {
		index = KineLabelIndex
		args = []any{listOpts.KindLabelSelector.Kind, listOpts.KindLabelSelector.Namespace, listOpts.KindLabelSelector.Name}
	}
	iter, err := txn.Get(table, index, args...)
	if err != nil {
		return nil, err
	}
	var objs []any
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		objs = append(objs, obj)
	}
	return objs, nil
}

// Delete methods
func (c *dbCache) DeleteRoute(r *Route) error {
	return c.delete("route", r)
}

func (c *dbCache) DeleteService(s *Service) error {
	return c.delete("service", s)
}

func (c *dbCache) DeleteUpstream(u *Upstream) error {
	return c.delete("upstream", u)
}

func (c *dbCache) DeleteSSL(ssl *SSL) error {
	return c.delete("ssl", ssl)
}

func (c *dbCache) DeleteGlobalRule(gr *GlobalRule) error {
	return c.delete("global_rule", gr)
}

func (c *dbCache) delete(table string, obj any) error {
	txn := c.db.Txn(true)
	defer txn.Abort()
	if err := txn.Delete(table, obj); err != nil {
		if err == memdb.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	txn.Commit()
	return nil
}

// =============================================================================
// DeepCopy Methods
// =============================================================================

// DeepCopy methods for kine types
func (r *Route) DeepCopy() *Route {
	if r == nil {
		return nil
	}
	copied := &Route{
		Metadata: copyMetadata(r.Metadata),
		URIs:     copyStringSlice(r.URIs),
		Methods:  copyMethods(r.Methods),
		Hosts:    copyStringSlice(r.Hosts),
		Priority: r.Priority,
		Plugins:  copyPlugins(r.Plugins),
		Upstream: r.Upstream.DeepCopy(),
		Timeout:  copyTimeout(r.Timeout),
	}
	if r.URI != nil {
		uri := *r.URI
		copied.URI = &uri
	}
	if r.Host != nil {
		host := *r.Host
		copied.Host = &host
	}
	if r.UpstreamID != nil {
		upstreamID := *r.UpstreamID
		copied.UpstreamID = &upstreamID
	}
	if r.ServiceID != nil {
		serviceID := *r.ServiceID
		copied.ServiceID = &serviceID
	}
	return copied
}

func (s *Service) DeepCopy() *Service {
	if s == nil {
		return nil
	}
	copied := &Service{
		Metadata: copyMetadata(s.Metadata),
		Plugins:  copyPlugins(s.Plugins),
		Upstream: s.Upstream.DeepCopy(),
		Hosts:    copyStringSlice(s.Hosts),
	}
	if s.UpstreamID != nil {
		upstreamID := *s.UpstreamID
		copied.UpstreamID = &upstreamID
	}
	return copied
}

func (u *Upstream) DeepCopy() *Upstream {
	if u == nil {
		return nil
	}
	copied := &Upstream{
		Metadata: copyMetadata(u.Metadata),
		Nodes:    copyNodes(u.Nodes),
		Type:     u.Type,
		Checks:   u.Checks.DeepCopy(),
		HashOn:   u.HashOn,
		Key:      u.Key,
		Scheme:   u.Scheme,
		PassHost: u.PassHost,
		Timeout:  copyTimeout(u.Timeout),
	}
	if u.Retries != nil {
		retries := *u.Retries
		copied.Retries = &retries
	}
	if u.RetryTimeout != nil {
		retryTimeout := *u.RetryTimeout
		copied.RetryTimeout = &retryTimeout
	}
	if u.UpstreamHost != nil {
		upstreamHost := *u.UpstreamHost
		copied.UpstreamHost = &upstreamHost
	}
	return copied
}

func (s *SSL) DeepCopy() *SSL {
	if s == nil {
		return nil
	}
	return &SSL{
		Metadata: copyMetadata(s.Metadata),
		Cert:     s.Cert,
		Key:      s.Key,
		SNIs:     copyStringSlice(s.SNIs),
	}
}

func (g *GlobalRule) DeepCopy() *GlobalRule {
	if g == nil {
		return nil
	}
	return &GlobalRule{
		ID:      g.ID,
		Plugins: copyPlugins(g.Plugins),
	}
}

func (h *HealthCheck) DeepCopy() *HealthCheck {
	if h == nil {
		return nil
	}
	return &HealthCheck{
		Active: h.Active.DeepCopy(),
	}
}

func (a *ActiveCheck) DeepCopy() *ActiveCheck {
	if a == nil {
		return nil
	}
	copied := &ActiveCheck{
		Type:                   a.Type,
		Timeout:                a.Timeout,
		HTTPPath:               a.HTTPPath,
		HTTPSVerifyCertificate: a.HTTPSVerifyCertificate,
		ReqHeaders:             copyStringSlice(a.ReqHeaders),
		Healthy:                a.Healthy.DeepCopy(),
		Unhealthy:              a.Unhealthy.DeepCopy(),
	}
	if a.Host != nil {
		host := *a.Host
		copied.Host = &host
	}
	if a.Port != nil {
		port := *a.Port
		copied.Port = &port
	}
	return copied
}

func (h *Health) DeepCopy() *Health {
	if h == nil {
		return nil
	}
	return &Health{
		Interval:     h.Interval,
		HTTPStatuses: copyUint32Slice(h.HTTPStatuses),
		Successes:    h.Successes,
	}
}

func (u *Unhealthy) DeepCopy() *Unhealthy {
	if u == nil {
		return nil
	}
	return &Unhealthy{
		HTTPFailures: u.HTTPFailures,
		TCPFailures:  u.TCPFailures,
	}
}

// Helper functions for deep copying
func copyMetadata(m adc.Metadata) adc.Metadata {
	return adc.Metadata{
		ID:     m.ID,
		Name:   m.Name,
		Desc:   m.Desc,
		Labels: copyLabels(m.Labels),
	}
}

func copyMethods(methods []Method) []Method {
	if methods == nil {
		return nil
	}
	copied := make([]Method, len(methods))
	copy(copied, methods)
	return copied
}

func copyPlugins(plugins map[string]any) map[string]any {
	if plugins == nil {
		return nil
	}
	// Note: This is a shallow copy of the map
	// For deep copy of plugin configs, we'd need to serialize/deserialize
	copied := make(map[string]any, len(plugins))
	for k, v := range plugins {
		copied[k] = v
	}
	return copied
}

func copyTimeout(t *Timeout) *Timeout {
	if t == nil {
		return nil
	}
	return &Timeout{
		Connect: t.Connect,
		Send:    t.Send,
		Read:    t.Read,
	}
}

func copyNodes(nodes map[string]uint32) map[string]uint32 {
	if nodes == nil {
		return nil
	}
	copied := make(map[string]uint32, len(nodes))
	for k, v := range nodes {
		copied[k] = v
	}
	return copied
}

func copyUint32Slice(slice []uint32) []uint32 {
	if slice == nil {
		return nil
	}
	copied := make([]uint32, len(slice))
	copy(copied, slice)
	return copied
}
