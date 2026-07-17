package catalog

const (
	BootstrapServiceArgoCD = "argo-cd"
	BootstrapServiceCRDs   = "bootstrap-crds"
)

var bootstrapServices = map[string]struct{}{
	BootstrapServiceArgoCD: {},
	BootstrapServiceCRDs:   {},
}

// IsBootstrapService reports whether the service is part of the implicit bootstrap foundation.
func IsBootstrapService(name string) bool {
	_, ok := bootstrapServices[name]
	return ok
}

// UserConfigurableServices returns a catalog view without bootstrap-only services.
func (c Catalog) UserConfigurableServices() Catalog {
	filtered := Catalog{Services: make(map[string]ServiceDefinition, len(c.Services))}
	for name, definition := range c.Services {
		if IsBootstrapService(name) {
			continue
		}
		filtered.Services[name] = definition
	}
	return filtered
}
