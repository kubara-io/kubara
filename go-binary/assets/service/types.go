package service

// Status is the desired state of a service.
type Status string

const (
	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
)

// Storage contains service storage settings.
type Storage struct {
	ClassName string `json:"className,omitempty" yaml:"className,omitempty" jsonschema:"title=Storage Class Name,description=Optional storage class name override for persistent volumes.,minLength=1"`
}

// Networking contains service networking settings.
type Networking struct {
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty" jsonschema:"title=Ingress Annotations,description=Optional ingress annotation overrides for this service."`
}

// Config holds arbitrary service-specific values.
type Config map[string]any

// Service represents the desired state and configuration of a service.
type Service struct {
	// Status defines the desired status for the service.
	// If not specified, the service will be disabled by default.
	Status     Status      `json:"status" yaml:"status" jsonschema:"title=Service Status,description=The desired status of the service.,enum=enabled,enum=disabled,default=disabled"`
	// Storage contains optional storage-related settings for the service.
	// These settings may be used to customize the generated storage interaction for the service, if applicable.
	Storage    *Storage    `json:"storage,omitempty" yaml:"storage,omitempty" jsonschema:"title=Storage Settings,description=Storage-related service settings."`
	// Networking contains optional networking-related settings for the service.
	// These settings may be used to customize the generated network settings for the service, if applicable.
	Networking *Networking `json:"networking,omitempty" yaml:"networking,omitempty" jsonschema:"title=Networking Settings,description=Networking-related service settings."`
	// Config contains arbitrary service-specific configuration values.
	// The schema for these values is defined in the catalog's service definition and enforced at runtime.
	Config     Config      `json:"config,omitempty" yaml:"config,omitempty" jsonschema:"title=Service Config,description=Service-specific configuration"`
}

// Services maps service names to service instances.
type Services map[string]Service

func (c Config) Clone() Config {
	if len(c) == 0 {
		return nil
	}
	out := make(Config, len(c))
	for k, v := range c {
		out[k] = cloneValue(v)
	}
	return out
}

func CloneValue(value any) any {
	return cloneValue(value)
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = cloneValue(v)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneValue(typed[i])
		}
		return out
	default:
		return value
	}
}
