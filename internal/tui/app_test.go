package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel()
	assert.Equal(t, ViewCases, m.state)
	assert.False(t, m.quitting)
	assert.Equal(t, 0, m.navCursor)
	assert.NotEmpty(t, m.llmProvider)
}

func TestModelTransitions(t *testing.T) {
	m := InitialModel()

	// Switch to sidebar focus
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.True(t, m2.(model).focusNav)

	// Move down to 'Analysis' (navCursor 1)
	m3, _ := m2.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.Equal(t, 1, m3.(model).navCursor)

	// Press enter to switch state
	m4, _ := m3.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, ViewAnalysis, m4.(model).state)
}

func TestQuit(t *testing.T) {
	m := InitialModel()
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	assert.True(t, m2.(model).quitting)
	assert.NotNil(t, cmd)
}

func TestNavFocus(t *testing.T) {
	m := InitialModel()
	assert.False(t, m.focusNav) // Default: sidebar not focused for arrows

	// Tab toggles focus
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.True(t, m2.(model).focusNav)

	m3, _ := m2.(model).Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.False(t, m3.(model).focusNav)
}

func TestSettingsLLMProviderCycle(t *testing.T) {
	m := InitialModel()
	m.state = ViewSettings
	m.settingsCursor = SettingLLMProvider

	initialProvider := m.llmProvider

	// Press enter to cycle provider
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2Model := m2.(model)

	assert.NotEqual(t, initialProvider, m2Model.llmProvider)
	assert.NotEmpty(t, m2Model.llmEndpoint)
	assert.NotEmpty(t, m2Model.modelName)
	assert.NotEmpty(t, m2Model.availableModels)
}

func TestSettingsLLMEditMode(t *testing.T) {
	m := InitialModel()
	m.state = ViewSettings
	m.settingsCursor = SettingLLMModel

	// Press 'e' to enter edit mode
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m2Model := m2.(model)
	assert.True(t, m2Model.isEditingSetting)
	assert.Equal(t, SettingLLMModel, m2Model.editSettingIndex)

	// Type custom model name
	m2Model.settingInput.SetValue("deepseek-r1:8b")

	// Press enter to save
	m3, _ := m2Model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3Model := m3.(model)
	assert.False(t, m3Model.isEditingSetting)
	assert.Equal(t, "deepseek-r1:8b", m3Model.modelName)
}

func TestSettingsConnectionStatus(t *testing.T) {
	m := InitialModel()
	m.state = ViewSettings

	// Send successful connection message
	m2, _ := m.Update(TestConnectionMsg{Success: true, Message: "Connected to ollama (12ms)"})
	m2Model := m2.(model)
	assert.Contains(t, m2Model.testStatus, "Connected to ollama")

	// Send error connection message
	m3, _ := m.Update(TestConnectionMsg{Success: false, Message: "Auth failed"})
	m3Model := m3.(model)
	assert.Contains(t, m3Model.testStatus, "Auth failed")
}

func TestFormatHostAndMaskAPIKey(t *testing.T) {
	assert.Equal(t, "localhost:11434", formatHost("http://localhost:11434/api/generate"))
	assert.Equal(t, "api.openai.com", formatHost("https://api.openai.com/v1"))
	assert.Equal(t, "localhost", formatHost(""))

	assert.Equal(t, "[Not set / None]", maskAPIKey(""))
	assert.Equal(t, "••••••••", maskAPIKey("short"))
	assert.Equal(t, "sk-••••1234", maskAPIKey("sk-abcdefgh1234"))
}
