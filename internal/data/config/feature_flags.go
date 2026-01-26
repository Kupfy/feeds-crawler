package config

import (
	"fmt"
	"strings"
)

type FeatureFlags map[string]bool

// Decode implements envconfig.Decoder interface
func (f *FeatureFlags) Decode(value string) error {
	*f = make(map[string]bool)

	if value == "" {
		return nil
	}

	// Parse comma-separated list: "llm_cleaned_html:true,llm_full_html:false"
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid feature flag format: %s (expected key:value)", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(strings.ToLower(parts[1]))

		if value == "true" || value == "1" || value == "yes" {
			(*f)[key] = true
		} else if value == "false" || value == "0" || value == "no" {
			(*f)[key] = false
		} else {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
	}

	return nil
}

func (f *FeatureFlags) IsEnabled(key string) bool {
	return (*f)[key]
}
