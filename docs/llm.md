# SPECTRE Intelligence Layer (AI) — Multi-Provider LLM Guide

SPECTRE is completely **LLM-independent**. Operators can seamlessly swap between local offline inference engines and cloud model providers depending on mission privacy, hardware resources, and operational needs.

---

## 1. Supported Providers

| Provider | Type | Default Endpoint | Recommended Models | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Ollama** | Local | `http://localhost:11434` | `llama3`, `mistral`, `qwen2.5:7b`, `llava` | Fully local & private. Offline operation with zero external data transfer. |
| **OpenAI** | Cloud | `https://api.openai.com/v1` | `gpt-4o-mini`, `gpt-4o`, `o3-mini` | High-accuracy cloud reasoning and synthesis. |
| **Anthropic** | Cloud | `https://api.anthropic.com/v1` | `claude-3-5-sonnet-20241022`, `claude-3-5-haiku-20241022` | Advanced OSINT analysis, entity synthesis, and long-context reasoning. |
| **Google Gemini** | Cloud | `https://generativelanguage.googleapis.com/v1beta/openai` | `gemini-1.5-flash`, `gemini-2.0-flash`, `gemini-1.5-pro` | Fast multimodal vision and document analysis. |
| **Groq** | Cloud | `https://api.groq.com/openai/v1` | `llama-3.3-70b-versatile`, `deepseek-r1-distill-llama-70b` | Ultra-fast LPUs for real-time investigation chat. |
| **OpenRouter** | Cloud Aggregator | `https://openrouter.ai/api/v1` | `meta-llama/llama-3.3-70b-instruct`, `anthropic/claude-3.5-sonnet` | Single API key access to hundreds of open and proprietary models. |
| **LM Studio** | Local | `http://localhost:1234/v1` | Any loaded GGUF (e.g. `llama-3.2-3b`, `qwen2.5-7b`) | GUI-driven local model runner with OpenAI-compatible API. |
| **LocalAI / vLLM** | Local / Self-Hosted | `http://localhost:8080/v1` | `gpt-4`, `llama3`, `mistral` | Self-hosted OpenAI-compatible server on local GPU clusters. |
| **Custom** | Custom | User-specified | User-specified | Any endpoint conforming to OpenAI chat completions or custom protocols. |

---

## 2. Interactive Configuration via the Modern Horizontal Console

You can configure, test, and swap LLM backends dynamically inside the **SPECTRE Interactive Console** using intuitive `?` commands directly from the chat prompt.

1. **Launch the interactive console:**
   ```bash
   spectre console
   # or simply:
   spectre
   ```

2. **Minimalist Header & Dynamic Model Identification:**
   - The top header displays strictly the application name on the left (**`✻ SPECTRE`**) and the active model on the right (**`Model: <model_name>`**).
   - If no model is configured, or if a cloud provider is selected whose API key is missing, SPECTRE dynamically detects this state and displays:
     ```
     ✻ SPECTRE                                                      Model: none
     ```

3. **Managing Models via `?` Commands:**
   - **View active model & presets:** Type `?model` to inspect the active model and available presets.
   - **Switch models dynamically:** Type `?model <name>` (e.g. `?model gpt-4o`, `?model llama3`, `?model none`). The top bar immediately synchronizes to reflect the change.
   - **Inspect all settings:** Type `?settings` to display current LLM provider, endpoint URL, API key status, Ghost Mode, and collector configurations.
   - **Live update configuration:** Type `?set <key> <value>`:
     ```text
     ?set llm.provider openai
     ?set llm.model gpt-4o-mini
     ?set llm.api_key sk-proj-...
     ```
   - **Test connectivity:** Type `?test` to verify authorization and latency to the active LLM endpoint (e.g. `✔ Connection Verified! Latency: 42ms`).
   - **Scan available models:** Type `?scan` to probe local Ollama / LM Studio or remote endpoints for all available models.

4. All configuration changes executed in the console are automatically saved to disk (`configs/default.yaml`).

---

## 3. Configuration via CLI

You can also update LLM settings via the command line:

```bash
# Switch to OpenAI
spectre config set llm.provider openai
spectre config set llm.url https://api.openai.com/v1
spectre config set llm.model gpt-4o-mini
spectre config set llm.api_key sk-proj-...

# Switch to Anthropic
spectre config set llm.provider anthropic
spectre config set llm.model claude-3-5-sonnet-20241022
spectre config set llm.api_key sk-ant-...

# Switch to Local LM Studio
spectre config set llm.provider lmstudio
spectre config set llm.url http://localhost:1234/v1
spectre config set llm.model local-model

# Switch back to Local Ollama
spectre config set llm.provider ollama
spectre config set llm.url http://localhost:11434
spectre config set llm.model llama3
```

---

## 4. Configuration via `configs/default.yaml`

Edit `configs/default.yaml`:

```yaml
# LLM Configuration
# Supported providers: "ollama", "openai", "anthropic", "gemini", "groq", "openrouter", "lmstudio", "localai", "custom"
llm:
  provider: "ollama"
  url: "http://localhost:11434" # Base URL (e.g. http://localhost:11434, https://api.openai.com/v1, http://localhost:1234/v1)
  model: "llama3"
  api_key: ""                   # Required for cloud providers (OpenAI, Anthropic, Gemini, Groq, OpenRouter)
  timeout: 120
  temperature: 0.7
  max_tokens: 4096
```

---

## 5. How It Works

1. **Synthesis (`spectre analyze`)**:
   - The Go engine collects and hashes target evidence records from the SQLite database.
   - It delegates analysis to the Python analyzer bridge with the active `LLMConfig`.
   - The Python layer dispatches a structured prompt to the selected LLM provider (using native JSON schemas or enforced JSON prompts), extracting findings, risks, and next steps.
2. **Interactive Autonomous Chat (`spectre chat` / TUI Chat)**:
   - Uses an agentic multi-turn loop.
   - Supports both native function calling (`tool_calls` / `tool_use`) and robust prompt-based tool execution across all model sizes.
3. **Question Answering (`spectre llm query`)**:
   - Takes case records as prompt context and responds to operator questions using evidence traceability.
4. **Visual Analysis (`spectre llm vision`)**:
   - Base64-encodes screenshot evidence and routes it to multi-modal vision models (`llava` on Ollama, `gpt-4o` on OpenAI, `claude-3-5-sonnet` on Anthropic, `gemini-1.5-flash` on Gemini).

---

## 6. Fallback Mode

If the selected LLM backend is offline or credentials fail:
- SPECTRE safely catches the failure without crashing.
- Returns a structured "Synthesis Unavailable" report outlining that manual review is required.
- In chat mode, identifies the failing provider name and provides helpful diagnostic guidance.
- The TUI status bar flags connection issues immediately when testing connectivity.
