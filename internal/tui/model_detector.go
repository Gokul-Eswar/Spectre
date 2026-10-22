package tui

import (
	"strings"

	"github.com/spf13/viper"
)

// DynamicModelStatus encapsulates detected model information.
type DynamicModelStatus struct {
	Name         string
	IsConfigured bool
	DisplayTag   string
}

// DetectActiveModel dynamically inspects runtime configuration and determines
// whether a valid model is chosen, or automatically resolves to "none".
func DetectActiveModel() DynamicModelStatus {
	rawModel := strings.TrimSpace(viper.GetString("llm.model"))
	rawProvider := strings.ToLower(strings.TrimSpace(viper.GetString("llm.provider")))
	apiKey := strings.TrimSpace(viper.GetString("llm.api_key"))

	if rawModel == "" || strings.EqualFold(rawModel, "none") {
		return DynamicModelStatus{
			Name:         "none",
			IsConfigured: false,
			DisplayTag:   "Model: none",
		}
	}

	// Providers that require an API key to function
	cloudProviders := map[string]bool{
		"openai":     true,
		"anthropic":  true,
		"gemini":     true,
		"groq":       true,
		"openrouter": true,
	}

	if cloudProviders[rawProvider] {
		// Also check keys.<provider> fallback
		if apiKey == "" {
			apiKey = strings.TrimSpace(viper.GetString("keys." + rawProvider))
		}
		if apiKey == "" {
			return DynamicModelStatus{
				Name:         "none",
				IsConfigured: false,
				DisplayTag:   "Model: none",
			}
		}
	}

	return DynamicModelStatus{
		Name:         rawModel,
		IsConfigured: true,
		DisplayTag:   "Model: " + rawModel,
	}
}

// SetActiveModel updates the active model and synchronizes configuration.
func SetActiveModel(modelName string) DynamicModelStatus {
	clean := strings.TrimSpace(modelName)
	if clean == "" || strings.EqualFold(clean, "none") {
		viper.Set("llm.model", "")
	} else {
		viper.Set("llm.model", clean)
	}
	return DetectActiveModel()
}
