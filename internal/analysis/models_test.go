package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSupportedProviders(t *testing.T) {
	providers := GetSupportedProviders()
	assert.Contains(t, providers, "ollama")
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
	assert.Contains(t, providers, "gemini")
	assert.Contains(t, providers, "groq")
	assert.Contains(t, providers, "openrouter")
	assert.Contains(t, providers, "lmstudio")
	assert.Contains(t, providers, "localai")
	assert.Contains(t, providers, "custom")
}

func TestGetPresetsForProvider(t *testing.T) {
	url, model, presets := GetPresetsForProvider("openai")
	assert.Equal(t, "https://api.openai.com/v1", url)
	assert.Equal(t, "gpt-4o-mini", model)
	assert.Contains(t, presets, "gpt-4o")

	url, model, presets = GetPresetsForProvider("anthropic")
	assert.Equal(t, "https://api.anthropic.com/v1", url)
	assert.Contains(t, presets, "claude-3-5-sonnet-20241022")

	url, model, presets = GetPresetsForProvider("ollama")
	assert.Equal(t, "http://localhost:11434", url)
	assert.Equal(t, "llama3", model)
	assert.Contains(t, presets, "llama3")
}

func TestFetchAvailableModelsFallback(t *testing.T) {
	// Without running backend, should fallback to presets
	models, err := FetchAvailableModels()
	assert.NoError(t, err)
	assert.NotEmpty(t, models)
}
