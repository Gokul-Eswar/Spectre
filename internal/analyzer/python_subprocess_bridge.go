package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// PythonCommand defines the command-line flags/arguments passed to Python.
// By default, we run "-m analyzer" which tells Python to run the 'analyzer' package as a module (looking for __main__.py).
// We use a slice of strings to make it easy to append CLI flags or override the target command during unit testing.
var PythonCommand = []string{"-m", "analyzer"}

// Request defines the structure sent to the Python analyzer subprocess via stdin or CLI argument.
// This struct maps directly to the expected JSON input schema on the Python side.
type Request struct {
	// Task specifies what operation Python should run (e.g. "synthesize", "chat", "query", "vision").
	Task string `json:"task"`
	// CaseID represents the unique identifier of the active investigation case.
	CaseID string `json:"case_id"`
	// CaseName is the human-readable name of the case.
	CaseName string `json:"case_name"`
	// Context passes general string context or raw data (like base64 images for vision tasks).
	Context string `json:"context"`
	// Model specifies which local LLM model to use (e.g. "llama3", "mistral", "llava").
	Model string `json:"model"`
	// Data holds arbitrary structured payload (like node/link data for visualization, or query strings).
	Data interface{} `json:"data"`
	// LLMConfig contains connection details, endpoints, and timeouts for hitting the LLM provider.
	LLMConfig LLMConfig `json:"llm_config"`
	// Messages is used for multi-turn conversations in chat mode, representing chat history.
	Messages []Message `json:"messages,omitempty"`
	// Tools defines the array of tool JSON schemas the LLM is allowed to invoke.
	Tools []interface{} `json:"tools,omitempty"`
}

// Message represents a single turn (user, assistant, or system prompt) in a conversational LLM interaction.
type Message struct {
	// Role defines who sent the message: "system", "user", or "assistant".
	Role string `json:"role"`
	// Content contains the raw text of the message.
	Content string `json:"content"`
}

// Response represents the structured response returned by the Python analyzer's stdout.
// The Go application reads and parses this structure to act on the LLM's decisions.
type Response struct {
	// Role defines the message sender (typically "assistant").
	Role string `json:"role"`
	// Content contains the text output or final answer from the LLM.
	Content string `json:"content"`
	// ToolUse is populated if the LLM decides it needs to invoke an external tool instead of replying in plain text.
	ToolUse *ToolUse `json:"tool_use,omitempty"`
	// Error contains any application-level error message reported from the Python side.
	Error string `json:"error,omitempty"`
}

// ToolUse holds the specific tool name and argument values requested by the LLM.
type ToolUse struct {
	// Name matches one of the registered tools (e.g., "collect", "search_evidence").
	Name string `json:"name"`
	// Arguments holds key-value pairs matching the parameters required by the tool.
	Arguments map[string]interface{} `json:"arguments"`
}

// Validate performs sanity checks on the request payload prior to spawning the Python process.
// This prevents resource wastage on invalid sub-tasks.
func (r *Request) Validate() error {
	// A task type must always be specified so Python knows which handler to dispatch.
	if r.Task == "" {
		return fmt.Errorf("task is required")
	}
	// Most tasks require an active CaseID to contextually resolve database records.
	// We allow 'visualize' to run without a CaseID if the payload contains self-contained visual data in the Data field.
	if r.CaseID == "" && r.Task != "visualize" {
		return fmt.Errorf("case_id is required")
	}
	return nil
}

// LLMConfig holds authorization and connection coordinates for local/remote LLM servers.
type LLMConfig struct {
	// Provider specifies the backend platform (e.g. "ollama", "openai", "anthropic", "gemini", "groq", "openrouter", "lmstudio", "localai", "custom").
	Provider string `json:"provider"`
	// URL points to the API endpoint (e.g., "http://localhost:11434", "https://api.openai.com/v1").
	URL string `json:"url"`
	// APIKey is used for authenticated backends.
	APIKey string `json:"api_key"`
	// Timeout controls how long we wait for the model to reply before breaking the request (in seconds).
	Timeout int `json:"timeout"`
	// Temperature controls sampling temperature (optional).
	Temperature float64 `json:"temperature,omitempty"`
	// MaxTokens limits output length (optional).
	MaxTokens int `json:"max_tokens,omitempty"`
}

// GetLLMConfig builds an LLMConfig from Viper, filling provider-specific defaults if values are omitted.
func GetLLMConfig() LLMConfig {
	provider := strings.ToLower(strings.TrimSpace(viper.GetString("llm.provider")))
	if provider == "" {
		provider = "ollama"
	}

	url := strings.TrimSpace(viper.GetString("llm.url"))
	if url == "" {
		url = GetDefaultURLForProvider(provider)
	}

	timeout := viper.GetInt("llm.timeout")
	if timeout <= 0 {
		timeout = 120
	}

	apiKey := strings.TrimSpace(viper.GetString("llm.api_key"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(viper.GetString("keys." + provider))
	}

	return LLMConfig{
		Provider:    provider,
		URL:         url,
		APIKey:      apiKey,
		Timeout:     timeout,
		Temperature: viper.GetFloat64("llm.temperature"),
		MaxTokens:   viper.GetInt("llm.max_tokens"),
	}
}

// GetDefaultURLForProvider returns standard default endpoints for each supported provider.
func GetDefaultURLForProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "lmstudio":
		return "http://localhost:1234/v1"
	case "localai":
		return "http://localhost:8080/v1"
	case "ollama":
		return "http://localhost:11434"
	default:
		return "http://localhost:11434"
	}
}

// GetDefaultModelForProvider returns the recommended starter model for each provider.
func GetDefaultModelForProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai":
		return "gpt-4o-mini"
	case "anthropic":
		return "claude-3-5-sonnet-20241022"
	case "gemini":
		return "gemini-1.5-flash"
	case "groq":
		return "llama-3.3-70b-versatile"
	case "openrouter":
		return "meta-llama/llama-3.3-70b-instruct"
	case "lmstudio":
		return "local-model"
	case "localai":
		return "gpt-4"
	case "ollama":
		return "llama3"
	default:
		return "llama3"
	}
}

// IndexEvidence triggers the Python analyzer to perform vector store ingestion of all evidence files for a case.
// It lists all files in the case's evidence folder and sends their file paths to Python to update the vector database.
func IndexEvidence(caseID string) error {
	// Define the path where raw evidence files are stored for the case.
	evidenceDir := fmt.Sprintf("evidence_storage/%s", caseID)

	// Read the list of files in the directory.
	files, err := os.ReadDir(evidenceDir)
	if err != nil {
		return fmt.Errorf("failed to read evidence directory: %w", err)
	}

	// Filter out directories and build an array of absolute or relative file paths to send to Python.
	var filePaths []string
	for _, f := range files {
		if !f.IsDir() {
			filePaths = append(filePaths, fmt.Sprintf("%s/%s", evidenceDir, f.Name()))
		}
	}

	// Prepare the request payload for the Python analyzer's index_evidence task.
	req := Request{
		Task:   "index_evidence",
		CaseID: caseID,
		Data: map[string]interface{}{
			"case_id": caseID,
			"files":   filePaths,
		},
	}

	// Dispatch the task using the active task runner.
	_, err = GlobalTaskRunner.Run(req)
	return err
}

// TaskRunner abstracts the execution of Python analysis tasks.
// This interface allows us to decouple the main Go logic from system process spawning,
// making it easy to register mock runners for unit testing.
type TaskRunner interface {
	Run(req Request) (string, error)
}

// DefaultTaskRunner is the production implementation of TaskRunner.
// It spawns Python as a real OS subprocess.
type DefaultTaskRunner struct{}

// Run implements the TaskRunner interface by invoking the subprocess handler.
func (d *DefaultTaskRunner) Run(req Request) (string, error) {
	return RunPythonTask(req)
}

// GlobalTaskRunner holds the active task execution engine.
// By default, it uses the OS process launcher (DefaultTaskRunner), but tests can substitute mock implementations.
var GlobalTaskRunner TaskRunner = &DefaultTaskRunner{}

// RunPythonTask handles the low-level lifecycle of spawning the Python subprocess.
// It marshals the request, sets up OS context timeouts, detects the correct Python binary,
// runs the command, and decodes stdout/stderr.
func RunPythonTask(req Request) (string, error) {
	// 1. Sanity check the request parameters before spawning a subprocess.
	if err := req.Validate(); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	log.Info().
		Str("task", req.Task).
		Str("case_id", req.CaseID).
		Msg("python_task_started")

	// 2. Serialize the request struct to JSON. This JSON string will be passed as a command-line argument.
	inputJSON, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	// 3. Create a execution context with a safety timeout of 3 minutes.
	// This ensures that if the LLM hangs or fails to respond, we don't leave zombie Python processes running forever.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 4. Resolve the correct Python interpreter executable path dynamically.
	// 4. Resolve analyzer command & base arguments dynamically.
	// If a frozen analyzer executable (e.g. spectre-analyzer.exe) is found,
	// it will be returned as execBin with empty baseArgs.
	// Otherwise, it returns the Python interpreter binary and baseArgs (e.g., ["-m", "analyzer"]).
	execBin, baseArgs := resolveAnalyzerCommand()

	// 5. Build the argument slice: <baseArgs...> --task <task> --input <json_data>
	args := append(baseArgs, "--task", req.Task, "--input", string(inputJSON))

	// Create the OS command with our timeout context.
	cmd := exec.CommandContext(ctx, execBin, args...)

	// Define buffers to capture output streams separately.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 6. Launch the subprocess and wait for execution to complete.
	err = cmd.Run()
	output := stdout.String()
	stderrStr := stderr.String()

	// 7. Check for runtime errors
	if err != nil {
		// Check if the process exited because the context timeout was exceeded.
		if ctx.Err() == context.DeadlineExceeded {
			log.Error().Err(err).Str("task", req.Task).Msg("python_task_timeout")
			return "", fmt.Errorf("python analysis timed out after 3 minutes")
		}

		// Try to parse a structured JSON error response from stdout.
		// Often, the python code catches exceptions internally and prints a clean {"error": "..."} JSON.
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(stdout.Bytes(), &errResp) == nil && errResp.Error != "" {
			return "", fmt.Errorf("python task '%s' failed: %s", req.Task, errResp.Error)
		}

		// If stdout did not contain structured JSON, fallback to returning the raw stderr stream.
		if stderrStr != "" {
			return "", fmt.Errorf("python execution error: %s", stderrStr)
		}

		return "", fmt.Errorf("python execution failed: %w", err)
	}

	// 8. If execution succeeded but stdout is empty, report an error.
	if output == "" {
		return "", fmt.Errorf("python task '%s' returned empty output", req.Task)
	}

	// 9. Ensure the stdout output matches our JSON-only API contract.
	if !json.Valid(stdout.Bytes()) {
		log.Error().Str("task", req.Task).Str("output", output).Msg("invalid_json_from_python")
		return "", fmt.Errorf("python task '%s' returned invalid JSON", req.Task)
	}

	log.Debug().Str("task", req.Task).Msg("python_task_completed")
	return output, nil
}

func resolveAnalyzerCommand() (string, []string) {
	// 0. If PythonCommand has been explicitly overridden from standard default (e.g. in unit tests),
	// use system python + overridden PythonCommand.
	isDefaultPythonCmd := len(PythonCommand) == 2 && PythonCommand[0] == "-m" && PythonCommand[1] == "analyzer"

	if !isDefaultPythonCmd {
		pyPath := resolvePythonExecutable()
		return pyPath, PythonCommand
	}

	// 1. Check SPECTRE_ANALYZER_BIN environment variable override
	if val := os.Getenv("SPECTRE_ANALYZER_BIN"); val != "" {
		if _, err := os.Stat(val); err == nil {
			return val, nil
		}
	}

	// 2. Check for frozen analyzer executable next to spectre.exe
	execPath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(execPath)

		// Check for directory bundle (dist/spectre-analyzer/spectre-analyzer.exe or spectre-analyzer/spectre-analyzer.exe)
		winDirExe := filepath.Join(dir, "spectre-analyzer", "spectre-analyzer.exe")
		if _, err := os.Stat(winDirExe); err == nil {
			return winDirExe, nil
		}
		unixDirExe := filepath.Join(dir, "spectre-analyzer", "spectre-analyzer")
		if _, err := os.Stat(unixDirExe); err == nil {
			return unixDirExe, nil
		}

		// Check for single file executable (spectre-analyzer.exe or spectre-analyzer)
		winExe := filepath.Join(dir, "spectre-analyzer.exe")
		if _, err := os.Stat(winExe); err == nil {
			return winExe, nil
		}
		unixExe := filepath.Join(dir, "spectre-analyzer")
		if _, err := os.Stat(unixExe); err == nil {
			return unixExe, nil
		}
	}

	// 3. Check PATH for standalone spectre-analyzer binary
	for _, binName := range []string{"spectre-analyzer.exe", "spectre-analyzer"} {
		if path, err := exec.LookPath(binName); err == nil {
			return path, nil
		}
	}

	// 4. Fallback to system Python + PythonCommand
	pyPath := resolvePythonExecutable()
	return pyPath, PythonCommand
}

func resolvePythonExecutable() string {
	// 1. Environment Variable Overrides
	for _, env := range []string{"PYTHON_BIN", "PYTHON_PATH", "PYTHONPATH"} {
		if val := os.Getenv(env); val != "" {
			if _, err := os.Stat(val); err == nil {
				return val
			}
		}
	}

	// 2. Viper User Config Override
	if cfgPath := viper.GetString("python.path"); cfgPath != "" {
		if _, err := os.Stat(cfgPath); err == nil {
			return cfgPath
		}
	}

	// 3. Walk up directory tree to locate .venv or venv from current working directory
	dir, err := os.Getwd()
	if err == nil {
		for i := 0; i < 5; i++ {
			winVenv := filepath.Join(dir, ".venv", "Scripts", "python.exe")
			if _, err := os.Stat(winVenv); err == nil {
				return winVenv
			}
			unixVenv := filepath.Join(dir, ".venv", "bin", "python")
			if _, err := os.Stat(unixVenv); err == nil {
				return unixVenv
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 4. System PATH Discovery (python3, python, py) - verify binary is functional
	for _, binary := range []string{"python3", "python", "py"} {
		if path, err := exec.LookPath(binary); err == nil {
			// Verify executable actually runs (to avoid Windows App Execution Alias traps)
			out, err := exec.Command(path, "--version").Output()
			if err == nil && len(out) > 0 {
				return path
			}
		}
	}

	return "python"
}
