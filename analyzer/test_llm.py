import pytest
import responses
import json
from analyzer.llm import (
    chat,
    analyze_case,
    query_case,
    analyze_image,
    generate_dorks,
    extract_json,
    detect_provider,
)


@responses.activate
def test_chat_success():
    # Mock Ollama response
    mock_response = {"message": {"content": "Hello! I am SPECTRE."}}
    responses.add(
        responses.POST,
        "http://localhost:11434/api/chat",
        json=mock_response,
        status=200,
    )

    data = {"messages": [{"role": "user", "content": "Hi"}], "model": "llama3"}

    result = chat(data)
    assert result["role"] == "assistant"
    assert result["content"] == "Hello! I am SPECTRE."


@responses.activate
def test_chat_tool_use():
    # Mock Ollama response with JSON tool call
    mock_response = {
        "message": {
            "content": '{"tool_use": {"name": "dns", "arguments": {"target": "google.com"}}}'
        }
    }
    responses.add(
        responses.POST,
        "http://localhost:11434/api/chat",
        json=mock_response,
        status=200,
    )

    data = {
        "messages": [{"role": "user", "content": "Scan google.com"}],
        "tools": [{"name": "dns"}],
    }

    result = chat(data)
    assert "tool_use" in result
    assert result["tool_use"]["name"] == "dns"


@responses.activate
def test_analyze_case_success():
    # Mock Ollama generate response
    mock_report = {
        "findings": ["Test finding"],
        "risks": ["Test risk"],
        "connections": [],
        "next_steps": [],
        "confidence": 0.9,
    }
    responses.add(
        responses.POST,
        "http://localhost:11434/api/generate",
        json={"response": json.dumps(mock_report)},
        status=200,
    )

    data = {
        "case_name": "Test Case",
        "context": "Some evidence data",
        "model": "llama3",
    }

    result = analyze_case(data)
    assert result["confidence"] == 0.9
    assert "Test finding" in result["findings"]


@responses.activate
def test_llm_failure_fallback():
    # Simulate a connection error
    responses.add(
        responses.POST,
        "http://localhost:11434/api/chat",
        body=Exception("Connection refused"),
    )

    data = {"messages": [{"role": "user", "content": "Hi"}]}

    result = chat(data)
    # Check that fallback mode responds to greetings
    assert "content" in result
    assert "Fallback" in result["content"]


@responses.activate
def test_chat_openai_compatible():
    # Mock OpenAI chat completions endpoint
    mock_response = {
        "choices": [
            {
                "message": {
                    "role": "assistant",
                    "content": "Hello from OpenAI-compatible provider!",
                }
            }
        ]
    }
    responses.add(
        responses.POST,
        "https://api.openai.com/v1/chat/completions",
        json=mock_response,
        status=200,
    )

    data = {
        "messages": [{"role": "user", "content": "Hello"}],
        "model": "gpt-4o-mini",
        "llm_config": {
            "provider": "openai",
            "url": "https://api.openai.com/v1",
            "api_key": "sk-test-key",
            "timeout": 60,
        },
    }

    result = chat(data)
    assert result["role"] == "assistant"
    assert result["content"] == "Hello from OpenAI-compatible provider!"
    # Ensure Authorization header was sent
    assert responses.calls[0].request.headers["Authorization"] == "Bearer sk-test-key"


@responses.activate
def test_chat_openai_native_tool_call():
    # Mock OpenAI native tool_calls
    mock_response = {
        "choices": [
            {
                "message": {
                    "role": "assistant",
                    "content": None,
                    "tool_calls": [
                        {
                            "id": "call_123",
                            "type": "function",
                            "function": {
                                "name": "whois",
                                "arguments": '{"target": "example.com"}',
                            },
                        }
                    ],
                }
            }
        ]
    }
    responses.add(
        responses.POST,
        "https://api.openai.com/v1/chat/completions",
        json=mock_response,
        status=200,
    )

    data = {
        "messages": [{"role": "user", "content": "Check whois"}],
        "model": "gpt-4o",
        "llm_config": {
            "provider": "openai",
            "url": "https://api.openai.com/v1",
            "api_key": "sk-test",
        },
    }

    result = chat(data)
    assert "tool_use" in result
    assert result["tool_use"]["name"] == "whois"
    assert result["tool_use"]["arguments"] == {"target": "example.com"}


@responses.activate
def test_chat_anthropic():
    # Mock Anthropic Messages endpoint
    mock_response = {
        "content": [
            {"type": "text", "text": "Hello from Claude!"}
        ]
    }
    responses.add(
        responses.POST,
        "https://api.anthropic.com/v1/messages",
        json=mock_response,
        status=200,
    )

    data = {
        "messages": [{"role": "user", "content": "Hello"}],
        "model": "claude-3-5-sonnet-20241022",
        "llm_config": {
            "provider": "anthropic",
            "url": "https://api.anthropic.com/v1",
            "api_key": "sk-ant-test",
        },
    }

    result = chat(data)
    assert result["role"] == "assistant"
    assert result["content"] == "Hello from Claude!"
    # Verify Anthropic headers
    assert responses.calls[0].request.headers["x-api-key"] == "sk-ant-test"
    assert responses.calls[0].request.headers["anthropic-version"] == "2023-06-01"


@responses.activate
def test_analyze_case_openai():
    mock_report = {
        "findings": ["OpenAI finding"],
        "risks": ["High risk asset exposed"],
        "connections": ["IP to domain match"],
        "next_steps": ["Isolate port 8080"],
        "missing_data": [],
        "suggested_collectors": ["ports"],
        "confidence": 0.95,
    }
    responses.add(
        responses.POST,
        "https://api.openai.com/v1/chat/completions",
        json={
            "choices": [
                {
                    "message": {
                        "content": f"```json\n{json.dumps(mock_report)}\n```"
                    }
                }
            ]
        },
        status=200,
    )

    data = {
        "case_name": "Cloud Case",
        "context": "Open ports: 8080, 22",
        "model": "gpt-4o",
        "llm_config": {
            "provider": "openai",
            "url": "https://api.openai.com/v1",
            "api_key": "sk-12345",
        },
    }

    result = analyze_case(data)
    assert result["confidence"] == 0.95
    assert "OpenAI finding" in result["findings"]


@responses.activate
def test_query_case_anthropic():
    responses.add(
        responses.POST,
        "https://api.anthropic.com/v1/messages",
        json={"content": [{"type": "text", "text": "The target domain expires in 2028."}]},
        status=200,
    )

    data = {
        "case_name": "Domain Check",
        "context": "Registrar info...",
        "data": "When does domain expire?",
        "model": "claude-3-5-sonnet-20241022",
        "llm_config": {
            "provider": "anthropic",
            "url": "https://api.anthropic.com",
            "api_key": "ant-key",
        },
    }

    result = query_case(data)
    assert "2028" in result["answer"]


@responses.activate
def test_analyze_image_openai():
    responses.add(
        responses.POST,
        "https://api.openai.com/v1/chat/completions",
        json={"choices": [{"message": {"content": "Login screen with admin portal."}}]},
        status=200,
    )

    data = {
        "data": "Analyze this screenshot",
        "context": "aGVsbG8=",
        "model": "gpt-4o",
        "llm_config": {
            "provider": "openai",
            "url": "https://api.openai.com/v1",
            "api_key": "sk-key",
        },
    }

    result = analyze_image(data)
    assert "Login screen" in result["answer"]


@responses.activate
def test_generate_dorks_groq():
    responses.add(
        responses.POST,
        "https://api.groq.com/openai/v1/chat/completions",
        json={
            "choices": [
                {
                    "message": {
                        "content": json.dumps({"dorks": ["site:target.com filetype:env", "site:target.com inurl:admin"]})
                    }
                }
            ]
        },
        status=200,
    )

    data = {
        "data": "target.com",
        "model": "llama-3.3-70b-versatile",
        "llm_config": {
            "provider": "groq",
            "url": "https://api.groq.com/openai/v1",
            "api_key": "gsk-key",
        },
    }

    result = generate_dorks(data)
    assert "dorks" in result
    assert len(result["dorks"]) == 2


def test_detect_provider():
    assert detect_provider({"provider": "openai"}) == "openai"
    assert detect_provider({"provider": "anthropic"}) == "anthropic"
    assert detect_provider({"url": "https://api.anthropic.com/v1"}) == "anthropic"
    assert detect_provider({"url": "https://api.groq.com/openai/v1"}) == "groq"
    assert detect_provider({"url": "http://localhost:1234/v1"}) == "lmstudio"
    assert detect_provider({"url": "http://localhost:11434"}) == "ollama"
    assert detect_provider({}) == "ollama"


def test_extract_json_varieties():
    # Plain JSON
    assert extract_json('{"key": "value"}') == {"key": "value"}
    # Markdown fence
    assert extract_json('```json\n{"hello": "world"}\n```') == {"hello": "world"}
    # Preamble + postamble
    assert extract_json('Sure, here is the result: {"status": "ok"} Thank you!') == {"status": "ok"}
    # Array
    assert extract_json('["item1", "item2"]') == ["item1", "item2"]
    # Invalid
    assert extract_json('Not a json at all') is None
