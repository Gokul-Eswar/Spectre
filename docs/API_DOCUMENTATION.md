# 🔌 Bridge API & Architecture Schema

This document details the internal architecture of **SPECTRE** and the Go-to-Python subprocess messaging contract.

---

## Architecture Overview

SPECTRE uses a hybrid architecture:
1.  **Core (Go)**: Performance-critical CLI/TUI logic, proxy orchestration, task coordination, database writes, and SQLite storage.
2.  **Analysis Layer (Python)**: Local LLM processing, vector embeddings storage (ChromaDB), and network graph generation.

```
┌─────────────────┐                 ┌────────────────────┐
│   Go Core       │  (JSON Envelope)│  Python Analyzer   │
│  (CLI / TUI)    ├────────────────►│    (Subprocess)    │
│                 │                 │                    │
│                 │◄────────────────┤                    │
└─────────────────┘  (JSON stdout)  └────────────────────┘
```

The Go core calls Python as an OS subprocess using `exec.CommandContext`. The Python subprocess runs in stateless module mode (`python.exe -m analyzer ...`), receives requests via a JSON envelope argument, and emits response payloads to stdout.

---

## 1. Subprocess Bridge Contract

### Execution Parameters
Go spawns the subprocess with the resolved Python interpreter (either `.venv/Scripts/python.exe` or the global system `python`):
```bash
python.exe -m analyzer --task <task_name> --input <json_envelope>
```

### JSON Request Envelope Schema
Every request from the Go bridge is formatted as a single JSON object matching the following structure:
```json
{
  "task": "query",
  "case_id": "a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f",
  "case_name": "suspect-infra-01",
  "context": "Raw case context evidence data concatenated as string...",
  "model": "llama3",
  "data": "A natural language query or task-specific parameter...",
  "llm_config": {
    "provider": "ollama",
    "url": "http://localhost:11434",
    "api_key": "optional-key-here",
    "timeout": 120
  }
}
```

### Supported Task Identifiers
*   `synthesize`: Runs case analysis context through local LLM to generate risks and next steps.
*   `query`: Runs prompt context query and returns raw conversational text answer.
*   `vision`: Encodes a local image file as base64 and posts to a multimodal model.
*   `index_evidence`: Ingests a list of evidence file paths into the ChromaDB vector database.
*   `search_evidence`: Queries ChromaDB semantic vectors and returns matching chunk strings.

---

## 2. Request and Response Schemas by Task

### Task: `query`
Used for conversational question answering.
*   **Request Data**: Natural language question string (e.g. `"What domains are active?"`).
*   **Response JSON**:
    ```json
    {
      "answer": "The active domain is target.com, resolving to 192.168.1.1..."
    }
    ```

### Task: `synthesize`
Used to generate comprehensive intelligence reports.
*   **Request Context**: Full list of evidence files, network details, and entity relationships.
*   **Response JSON**:
    ```json
    {
      "findings": ["Discovered staging subdomain dev.target.com", "Open SSH port 22 exposed"],
      "risks": ["Critical vulnerability on exposed SSH port", "Unsecured domain records"],
      "connections": ["Staging subdomain links to production host IP"],
      "next_steps": ["Recommend closing SSH port 22", "Conduct sub-domain asset search"],
      "confidence": 0.85
    }
    ```

### Task: `search_evidence`
Used for semantic lookup inside ChromaDB.
*   **Request Data**: Map containing `case_id` and search `query`.
*   **Response JSON**:
    ```json
    {
      "results": [
        {
          "id": "evidence-file-hash-01",
          "content": "Raw matching evidence content snippet..."
        }
      ]
    }
    ```

---

## 3. Go Core Packages (`internal/`)

*   **`internal/agent/`**:
    *   `autonomous_agent_loop.go`: Logic loop for the interactive AI session. Evaluates LLM text, registers tool calls, and routes execution.
    *   `agent_tool_registry.go`: Tools exposed to the LLM agent (`collect`, `get_case_summary`, `search_entities`, etc.).
*   **`internal/analyzer/`**:
    *   `python_subprocess_bridge.go`: Subprocess wrapper that serializes Go structs, locates Python env path, handles standard streams, and deserializes stdout.
*   **`internal/collector/`**:
    *   `registry.go`: Central loader for native (Go) and external (Python/Bash) collectors.
    *   `external_plugin_runner.go`: Wrapper that parses manifest files and executes external files.
*   **`internal/storage/`**:
    *   `database.go`: SQLite connection lifecycle. Handles schema migrations and thread-safe writes.
    *   `models.go`: Case, Entity, Relationship, and Evidence definitions.

---

## 4. Python Analyzer Modules (`analyzer/`)

*   **`__main__.py`**: Spawns command-line parser. Validates task targets and routes inputs to respective handlers.
*   **`llm.py`**: Interacts with local and cloud LLM APIs (Ollama, OpenAI, Anthropic, Gemini, Groq, OpenRouter, LM Studio, etc.). Handles request construction, timeout, and connection errors by returning clean fallback JSON buffers.
*   **`vector_store.py`**: Interacts with ChromaDB. Calculates local embeddings using `sentence-transformers` and runs similarity index lookups.
*   **`graph_viz.py`**: Connects entities, formats network coordinates, and creates PyVis templates.
