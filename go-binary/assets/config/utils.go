package config

import "fmt"

func toStringMap(v any) (map[string]any, error) {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, val := range typed {
			out[key] = normalizeValue(val)
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(typed))
		for k, val := range typed {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string map key %T", k)
			}
			out[key] = normalizeValue(val)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected object, got %T", v)
	}
}

func normalizeValue(v any) any {
	switch typed := v.(type) {
	case map[string]any, map[any]any:
		m, err := toStringMap(typed)
		if err != nil {
			return v
		}
		return m
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = normalizeValue(typed[i])
		}
		return out
	default:
		return v
	}
}
