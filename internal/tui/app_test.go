package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel()
	assert.False(t, m.quitting)
	assert.NotEmpty(t, m.modelStatus.Name)
	assert.NotNil(t, m.input)
	assert.NotNil(t, m.viewport)
}

func TestQuit(t *testing.T) {
	m := InitialModel()
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, m2.(model).quitting)
	assert.NotNil(t, cmd)
}

func TestDetectActiveModel_Configured(t *testing.T) {
	viper.Set("llm.provider", "ollama")
	viper.Set("llm.model", "llama3")

	status := DetectActiveModel()
	assert.True(t, status.IsConfigured)
	assert.Equal(t, "llama3", status.Name)
	assert.Equal(t, "Model: llama3", status.DisplayTag)
}

func TestDetectActiveModel_NoneFallback(t *testing.T) {
	viper.Set("llm.model", "")
	status := DetectActiveModel()
	assert.False(t, status.IsConfigured)
	assert.Equal(t, "none", status.Name)

	viper.Set("llm.model", "none")
	status2 := DetectActiveModel()
	assert.False(t, status2.IsConfigured)
	assert.Equal(t, "none", status2.Name)
}

func TestDetectActiveModel_MissingCloudKey(t *testing.T) {
	viper.Set("llm.provider", "openai")
	viper.Set("llm.model", "gpt-4o")
	viper.Set("llm.api_key", "")
	viper.Set("keys.openai", "")

	status := DetectActiveModel()
	assert.False(t, status.IsConfigured)
	assert.Equal(t, "none", status.Name)
}

func TestExecuteQuestionCommand_Help(t *testing.T) {
	res := ExecuteQuestionCommand("?", "")
	assert.False(t, res.IsError)
	assert.Contains(t, res.Output, "SPECTRE COMMAND DIRECTORY")
	assert.Contains(t, res.Output, "?cases")
	assert.Contains(t, res.Output, "?analyze")
}

func TestExecuteQuestionCommand_ModelSwitch(t *testing.T) {
	viper.Set("llm.provider", "ollama")
	res := ExecuteQuestionCommand("?model llama3-70b", "")
	assert.False(t, res.IsError)
	assert.True(t, res.ModelChange)
	assert.Contains(t, res.Output, "llama3-70b")

	// Set back to none
	res2 := ExecuteQuestionCommand("?model none", "")
	assert.True(t, res2.ModelChange)
	assert.Equal(t, "none", DetectActiveModel().Name)
}

func TestExecuteQuestionCommand_Unknown(t *testing.T) {
	res := ExecuteQuestionCommand("?unknownxyz", "")
	assert.True(t, res.IsError)
	assert.Contains(t, res.Output, "Unknown command")
}

func TestHorizontalResizeResponsiveness(t *testing.T) {
	m := InitialModel()
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mod := newModel.(model)

	assert.Equal(t, 120, mod.width)
	assert.Equal(t, 40, mod.height)
	assert.Equal(t, 118, mod.viewport.Width)
	assert.Equal(t, 114, mod.input.Width)
}

func TestFormatHostAndMaskAPIKey(t *testing.T) {
	assert.Equal(t, "localhost:11434", formatHost("http://localhost:11434/api/generate"))
	assert.Equal(t, "api.openai.com", formatHost("https://api.openai.com/v1"))
	assert.Equal(t, "localhost", formatHost(""))

	assert.Equal(t, "[Not set / None]", maskAPIKey(""))
	assert.Equal(t, "••••••••", maskAPIKey("short"))
	assert.Equal(t, "sk-••••1234", maskAPIKey("sk-abcdefgh1234"))
}
