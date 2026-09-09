import pytest
import responses
import json
from analyzer.llm import chat, analyze_case


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
