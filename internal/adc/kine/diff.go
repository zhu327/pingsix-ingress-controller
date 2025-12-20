package kine

import (
	"fmt"
	"sort"

	"github.com/google/go-cmp/cmp"

	"github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
)

// EventType represents the type of change event
type EventType string

const (
	EventTypeCreate EventType = "CREATE"
	EventTypeUpdate EventType = "UPDATE"
	EventTypeDelete EventType = "DELETE"
)

// ResourceType represents the type of Kine resource
type ResourceType string

const (
	ResourceTypeRoute      ResourceType = "route"
	ResourceTypeService    ResourceType = "service"
	ResourceTypeUpstream   ResourceType = "upstream"
	ResourceTypeSSL        ResourceType = "ssl"
	ResourceTypeGlobalRule ResourceType = "global_rule"
)

// Event represents a change event for a resource
type Event struct {
	Type         EventType    `json:"type"`
	ResourceType ResourceType `json:"resourceType"`
	ResourceID   string       `json:"resourceId"`
	ResourceName string       `json:"resourceName"`
	ParentID     string       `json:"parentId,omitempty"`
	OldValue     any          `json:"oldValue,omitempty"`
	NewValue     any          `json:"newValue,omitempty"`
}

// DiffOptions contains options for diff operation
type DiffOptions struct {
	Labels map[string]string
	Types  []string
}

// Differ interface for comparing resources and generating events
type Differ interface {
	// Diff compares resources and generates events
	Diff(newResources *TransferredResources, opts *DiffOptions) ([]Event, error)
}

// TransferredResources contains all transferred Kine resources
type TransferredResources struct {
	Routes      []*Route
	Services    []*Service
	SSLs        []*SSL
	GlobalRules []*GlobalRule
}

// differ implements the Differ interface
type differ struct {
	cache Cache
}

// NewDiffer creates a new Differ instance
func NewDiffer(cache Cache) Differ {
	return &differ{
		cache: cache,
	}
}

// Diff compares resources and generates events
func (d *differ) Diff(newResources *TransferredResources, opts *DiffOptions) ([]Event, error) {
	var events []Event

	// Filter resource types to diff
	typesToDiff := make(map[string]bool)
	if len(opts.Types) > 0 {
		for _, t := range opts.Types {
			typesToDiff[t] = true
		}
	}

	// Build KindSelector from labels if provided
	var listOpts []ListOption
	if len(opts.Labels) > 0 {
		kindSelector := &KindLabelSelector{
			Kind:      opts.Labels[label.LabelKind],
			Namespace: opts.Labels[label.LabelNamespace],
			Name:      opts.Labels[label.LabelName],
		}
		listOpts = append(listOpts, kindSelector)
	}

	// Diff routes
	if len(typesToDiff) == 0 || typesToDiff[string(ResourceTypeRoute)] {
		routeEvents, err := d.diffRoutes(newResources.Routes, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to diff routes: %w", err)
		}
		events = append(events, routeEvents...)
	}

	// Diff services
	if len(typesToDiff) == 0 || typesToDiff[string(ResourceTypeService)] {
		serviceEvents, err := d.diffServices(newResources.Services, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to diff services: %w", err)
		}
		events = append(events, serviceEvents...)
	}

	// Diff SSLs
	if len(typesToDiff) == 0 || typesToDiff[string(ResourceTypeSSL)] {
		sslEvents, err := d.diffSSLs(newResources.SSLs, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to diff ssls: %w", err)
		}
		events = append(events, sslEvents...)
	}

	// Diff global rules
	if len(typesToDiff) == 0 || typesToDiff[string(ResourceTypeGlobalRule)] {
		globalRuleEvents, err := d.diffGlobalRules(newResources.GlobalRules, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to diff global rules: %w", err)
		}
		events = append(events, globalRuleEvents...)
	}

	// Sort events by execution order
	sortEvents(events)

	return events, nil
}

// diffRoutes compares new routes with cached routes
func (d *differ) diffRoutes(newRoutes []*Route, listOpts []ListOption) ([]Event, error) {
	// Get cached routes
	cachedRoutes, err := d.cache.ListRoutes(listOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to list cached routes: %w", err)
	}

	// Build maps for comparison
	newMap := make(map[string]*Route)
	for _, route := range newRoutes {
		newMap[route.ID] = route
	}

	cachedMap := make(map[string]*Route)
	for _, route := range cachedRoutes {
		cachedMap[route.ID] = route
	}

	var events []Event

	// Find CREATE and UPDATE events
	for id, newRoute := range newMap {
		if cachedRoute, exists := cachedMap[id]; exists {
			// Check if update is needed
			if !areRoutesEqual(cachedRoute, newRoute) {
				events = append(events, Event{
					Type:         EventTypeUpdate,
					ResourceType: ResourceTypeRoute,
					ResourceID:   id,
					ResourceName: newRoute.Name,
					OldValue:     cachedRoute,
					NewValue:     newRoute,
				})
			}
		} else {
			// Create new route
			events = append(events, Event{
				Type:         EventTypeCreate,
				ResourceType: ResourceTypeRoute,
				ResourceID:   id,
				ResourceName: newRoute.Name,
				NewValue:     newRoute,
			})
		}
	}

	// Find DELETE events
	for id, cachedRoute := range cachedMap {
		if _, exists := newMap[id]; !exists {
			events = append(events, Event{
				Type:         EventTypeDelete,
				ResourceType: ResourceTypeRoute,
				ResourceID:   id,
				ResourceName: cachedRoute.Name,
				OldValue:     cachedRoute,
			})
		}
	}

	return events, nil
}

// diffServices compares new services with cached services
func (d *differ) diffServices(newServices []*Service, listOpts []ListOption) ([]Event, error) {
	// Get cached services
	cachedServices, err := d.cache.ListServices(listOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to list cached services: %w", err)
	}

	// Build maps for comparison
	newMap := make(map[string]*Service)
	for _, service := range newServices {
		newMap[service.ID] = service
	}

	cachedMap := make(map[string]*Service)
	for _, service := range cachedServices {
		cachedMap[service.ID] = service
	}

	var events []Event

	// Find CREATE and UPDATE events
	for id, newService := range newMap {
		if cachedService, exists := cachedMap[id]; exists {
			// Check if update is needed
			if !areServicesEqual(cachedService, newService) {
				events = append(events, Event{
					Type:         EventTypeUpdate,
					ResourceType: ResourceTypeService,
					ResourceID:   id,
					ResourceName: newService.Name,
					OldValue:     cachedService,
					NewValue:     newService,
				})
			}
		} else {
			// Create new service
			events = append(events, Event{
				Type:         EventTypeCreate,
				ResourceType: ResourceTypeService,
				ResourceID:   id,
				ResourceName: newService.Name,
				NewValue:     newService,
			})
		}
	}

	// Find DELETE events
	for id, cachedService := range cachedMap {
		if _, exists := newMap[id]; !exists {
			events = append(events, Event{
				Type:         EventTypeDelete,
				ResourceType: ResourceTypeService,
				ResourceID:   id,
				ResourceName: cachedService.Name,
				OldValue:     cachedService,
			})
		}
	}

	return events, nil
}

// diffSSLs compares new SSLs with cached SSLs
func (d *differ) diffSSLs(newSSLs []*SSL, listOpts []ListOption) ([]Event, error) {
	// Get cached SSLs
	cachedSSLs, err := d.cache.ListSSL(listOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to list cached ssls: %w", err)
	}

	// Build maps for comparison
	newMap := make(map[string]*SSL)
	for _, ssl := range newSSLs {
		newMap[ssl.ID] = ssl
	}

	cachedMap := make(map[string]*SSL)
	for _, ssl := range cachedSSLs {
		cachedMap[ssl.ID] = ssl
	}

	var events []Event

	// Find CREATE and UPDATE events
	for id, newSSL := range newMap {
		if cachedSSL, exists := cachedMap[id]; exists {
			// Check if update is needed
			if !areSSLsEqual(cachedSSL, newSSL) {
				events = append(events, Event{
					Type:         EventTypeUpdate,
					ResourceType: ResourceTypeSSL,
					ResourceID:   id,
					ResourceName: newSSL.Name,
					OldValue:     cachedSSL,
					NewValue:     newSSL,
				})
			}
		} else {
			// Create new SSL
			events = append(events, Event{
				Type:         EventTypeCreate,
				ResourceType: ResourceTypeSSL,
				ResourceID:   id,
				ResourceName: newSSL.Name,
				NewValue:     newSSL,
			})
		}
	}

	// Find DELETE events
	for id, cachedSSL := range cachedMap {
		if _, exists := newMap[id]; !exists {
			events = append(events, Event{
				Type:         EventTypeDelete,
				ResourceType: ResourceTypeSSL,
				ResourceID:   id,
				ResourceName: cachedSSL.Name,
				OldValue:     cachedSSL,
			})
		}
	}

	return events, nil
}

// diffGlobalRules compares new global rules with cached global rules
func (d *differ) diffGlobalRules(newGlobalRules []*GlobalRule, _ []ListOption) ([]Event, error) {
	// Get cached global rules - note: global rules don't support label filtering
	cachedGlobalRules, err := d.cache.ListGlobalRules()
	if err != nil {
		return nil, fmt.Errorf("failed to list cached global rules: %w", err)
	}

	// Build maps for comparison
	newMap := make(map[string]*GlobalRule)
	for _, rule := range newGlobalRules {
		newMap[rule.ID] = rule
	}

	cachedMap := make(map[string]*GlobalRule)
	for _, rule := range cachedGlobalRules {
		cachedMap[rule.ID] = rule
	}

	var events []Event

	// Find CREATE and UPDATE events
	for id, newRule := range newMap {
		if cachedRule, exists := cachedMap[id]; exists {
			// Check if update is needed
			if !areGlobalRulesEqual(cachedRule, newRule) {
				events = append(events, Event{
					Type:         EventTypeUpdate,
					ResourceType: ResourceTypeGlobalRule,
					ResourceID:   id,
					ResourceName: id, // GlobalRule uses ID as name
					OldValue:     cachedRule,
					NewValue:     newRule,
				})
			}
		} else {
			// Create new global rule
			events = append(events, Event{
				Type:         EventTypeCreate,
				ResourceType: ResourceTypeGlobalRule,
				ResourceID:   id,
				ResourceName: id,
				NewValue:     newRule,
			})
		}
	}

	// Find DELETE events
	for id, cachedRule := range cachedMap {
		if _, exists := newMap[id]; !exists {
			events = append(events, Event{
				Type:         EventTypeDelete,
				ResourceType: ResourceTypeGlobalRule,
				ResourceID:   id,
				ResourceName: id,
				OldValue:     cachedRule,
			})
		}
	}

	return events, nil
}

// Comparison functions for different resource types

// areRoutesEqual compares two routes for equality using go-cmp
func areRoutesEqual(a, b *Route) bool {
	return cmp.Equal(a, b)
}

// areServicesEqual compares two services for equality using go-cmp
func areServicesEqual(a, b *Service) bool {
	return cmp.Equal(a, b)
}

// areSSLsEqual compares two SSLs for equality using go-cmp
func areSSLsEqual(a, b *SSL) bool {
	return cmp.Equal(a, b)
}

// areGlobalRulesEqual compares two global rules for equality using go-cmp
func areGlobalRulesEqual(a, b *GlobalRule) bool {
	return cmp.Equal(a, b)
}

// sortEvents sorts events by execution order
// Order:
// 1. DELETE events (reverse dependency order: Route -> Service -> SSL -> GlobalRule)
// 2. UPDATE events (same as DELETE order: Route -> Service -> SSL -> GlobalRule)
// 3. CREATE events (forward dependency order: GlobalRule -> SSL -> Service -> Route)
func sortEvents(events []Event) {
	// Define order priority for each resource type
	// DELETE and UPDATE use the same order (reverse dependency order)
	deleteUpdateOrder := map[ResourceType]int{
		ResourceTypeRoute:      0,
		ResourceTypeService:    1,
		ResourceTypeSSL:        2,
		ResourceTypeGlobalRule: 3,
	}

	createOrder := map[ResourceType]int{
		ResourceTypeGlobalRule: 0,
		ResourceTypeSSL:        1,
		ResourceTypeService:    2,
		ResourceTypeRoute:      3,
	}

	sort.Slice(events, func(i, j int) bool {
		ei, ej := events[i], events[j]

		// First sort by event type: DELETE < UPDATE < CREATE
		if ei.Type != ej.Type {
			return eventTypePriority(ei.Type) < eventTypePriority(ej.Type)
		}

		// Within same event type, sort by resource type
		if ei.Type == EventTypeDelete || ei.Type == EventTypeUpdate {
			return deleteUpdateOrder[ei.ResourceType] < deleteUpdateOrder[ej.ResourceType]
		}
		if ei.Type == EventTypeCreate {
			return createOrder[ei.ResourceType] < createOrder[ej.ResourceType]
		}

		// Default: maintain original order (stable sort)
		return false
	})
}

// eventTypePriority returns the priority of an event type
func eventTypePriority(et EventType) int {
	switch et {
	case EventTypeDelete:
		return 0
	case EventTypeUpdate:
		return 1
	case EventTypeCreate:
		return 2
	default:
		return 3
	}
}

// TransferResources converts ADC resources to Kine resources
func TransferResources(resources *adc.Resources) (*TransferredResources, error) {
	result := &TransferredResources{}

	// Transfer services (which includes routes and upstream)
	for _, adcService := range resources.Services {
		kineService, kineRoutes, err := TransferService(adcService)
		if err != nil {
			return nil, fmt.Errorf("failed to transfer service %s: %w", adcService.Name, err)
		}
		if kineService != nil {
			result.Services = append(result.Services, kineService)
		}
		result.Routes = append(result.Routes, kineRoutes...)
	}

	// Transfer SSLs
	for _, adcSSL := range resources.SSLs {
		kineSSLs, err := TransferSSL(adcSSL)
		if err != nil {
			return nil, fmt.Errorf("failed to transfer ssl %s: %w", adcSSL.Name, err)
		}
		result.SSLs = append(result.SSLs, kineSSLs...)
	}

	// Transfer global rules
	if len(resources.GlobalRules) > 0 {
		kineGlobalRules := TransferGlobalRule(resources.GlobalRules)
		result.GlobalRules = append(result.GlobalRules, kineGlobalRules...)
	}

	return result, nil
}
