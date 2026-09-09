import requests
import json
import re
import time
import sys

# Global HTTP session object to enable connection pooling.
# Reusing TCP connections across multiple HTTP requests to local LLM engines (like Ollama)
# reduces latency overhead and socket consumption significantly.
session = requests.Session()


def extract_json(text):
    """
    Robustly extracts the first valid JSON object enclosed in curly braces {} from a string using regex.
    This is extremely helpful because local LLMs often output conversational text (e.g. "Sure, here is the JSON:")
    before or after the actual JSON structure.

    Parameters:
    - text (str): The raw text response from the LLM.

    Returns:
    - dict/list or None: Parsed JSON content if successful, or None if no valid JSON structure is found.
    """
    if not text:
        return None
    text = text.strip()
    # Regex searches for the first '{' and the last '}' spanning across multiple lines (re.DOTALL)
    match = re.search(r"\{.*\}", text, re.DOTALL)
    if match:
        try:
            return json.loads(match.group())
        except json.JSONDecodeError:
            pass
    return None


def chat(data):
    """
    Orchestrates an interactive chat turn between the user and the SPECTRE agent.
    If the agent needs to invoke a system tool, it returns a structured JSON payload with 'tool_use'.

    Parameters:
    - data (dict): Request dictionary containing:
      - messages (list): The list of conversational message history dictionaries.
      - tools (list): The list of JSON-schema dictionaries of allowed tools.
      - model (str): Model identifier (e.g. "llama3").
      - llm_config (dict): Connection parameters for the local LLM server.

    Returns:
    - dict: A dictionary with the message role and content or tool_use definition.
    """
    messages = data.get("messages", [])
    tools = data.get("tools", [])
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

    # Default to local Ollama chat API endpoint if not explicitly overridden.
    api_url = llm_config.get("url", "http://localhost:11434/api/chat")
    api_key = llm_config.get("api_key", "")
    timeout = llm_config.get("timeout", 120)

    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    # Define system instructions outlining the agent persona, role, and the JSON format for calling tools.
    system_msg = {
        "role": "system",
        "content": (
            "You are SPECTRE Agent, an OSINT automation assistant. "
            "You help users gather intelligence by using available tools. "
            "When you need to use a tool, respond with ONLY a JSON object in this format:\n"
            '{"tool_use": {"name": "tool_name", "arguments": { ... }}}\n'
            "Available tools:\n" + json.dumps(tools, indent=2) + "\n"
            "If you have enough information to answer the user, just provide a normal text response."
        ),
    }

    # Ensure the system prompt instructions are injected as the very first message.
    if not messages or messages[0].get("role") != "system":
        messages.insert(0, system_msg)

    # Set up the request payload for the Ollama /api/chat contract.
    payload = {"model": model, "messages": messages, "stream": False}

    try:
        # Send post request to the LLM backend.
        resp = session.post(api_url, json=payload, headers=headers, timeout=timeout)
        resp.raise_for_status()

        response_json = resp.json()

        # Normalize and extract the generated text based on standard API provider response envelopes.
        content = ""
        if "message" in response_json:
            # Standard Ollama /api/chat structure
            content = response_json["message"].get("content", "")
        elif "choices" in response_json:
            # Standard OpenAI chat completion structure
            content = response_json["choices"][0]["message"].get("content", "")
        elif "response" in response_json:
            # Simple generate completion structure
            content = response_json["response"]

        # Parse output to see if the model attempted to invoke a tool.
        tool_call = extract_json(content)
        if tool_call and "tool_use" in tool_call:
            return {"role": "assistant", "tool_use": tool_call["tool_use"]}

        # If no tool calls were requested, return the conversational text.
        return {"role": "assistant", "content": content}

    except Exception as e:
        # Fallback mechanism if the local LLM server is unreachable or offline.
        # We look at the last message to see if we can reply with a standard friendly offline banner.
        last_user_msg = ""
        for m in reversed(messages):
            if m.get("role") == "user":
                last_user_msg = m.get("content", "").lower()
                break

        if "hello" in last_user_msg or "hi" in last_user_msg:
            return {
                "role": "assistant",
                "content": "Hello! (Fallback Mode) I'm currently unable to reach my LLM backend, but I can still help with basic commands.",
            }

        return {
            "role": "assistant",
            "content": f"I'm sorry, I encountered an error and the LLM is unavailable: {str(e)}. Please check if Ollama is running.",
        }


def analyze_case(data):
    """
    Synthesizes and consolidates gathered intelligence records into a structured findings report.
    Forces JSON-only model formatting.

    Parameters:
    - data (dict): Input dictionary containing case details and raw target context summaries.

    Returns:
    - dict: A structured JSON output showing findings, risks, next actions, and suggested collectors.
    """
    case_name = data.get("case_name", "Unknown")
    context = data.get("context", "")
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

    api_url = llm_config.get("url", "http://localhost:11434/api/generate")
    api_key = llm_config.get("api_key", "")
    timeout = llm_config.get("timeout", 120)

    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    # Enforce strict JSON output schema and explain the required fields.
    system_prompt = (
        "You are SPECTRE, an expert intelligence analyst. "
        "Analyze the provided case data and generate a structured report. "
        "Your output MUST be strict JSON.\n"
        "Format:\n"
        "{\n"
        '  "findings": ["string"],\n'
        '  "risks": ["string"],\n'
        '  "connections": ["string"],\n'
        '  "next_steps": ["string"],\n'
        '  "missing_data": ["string"],\n'
        '  "suggested_collectors": ["string"],\n'
        '  "confidence": 0.85\n'
        "}"
    )

    full_prompt = f"{system_prompt}\n\nCASE DATA:\n{context}"

    payload = {"model": model, "prompt": full_prompt, "stream": False}

    # Implement a basic retry mechanism (up to 3 attempts) in case the local LLM outputs
    # corrupt or incomplete JSON blocks.
    retries = 3
    last_error = ""

    for attempt in range(retries):
        try:
            resp = session.post(api_url, json=payload, headers=headers, timeout=timeout)
            resp.raise_for_status()

            response_json = resp.json()
            raw_response = response_json.get("response", "")
            if not raw_response and "choices" in response_json:
                raw_response = response_json["choices"][0]["message"]["content"]

            # Verify and extract the JSON output.
            extracted = extract_json(raw_response)
            if extracted:
                return extracted

            last_error = "No valid JSON found in LLM response"

        except Exception as e:
            last_error = str(e)

        if attempt < retries - 1:
            time.sleep(1)  # Wait 1 second before retrying

    # Final fallback dictionary structure returned on failure.
    return {
        "findings": ["LLM synthesis unavailable. Raw data review required."],
        "risks": ["Unable to perform AI-assisted risk assessment."],
        "connections": [],
        "next_steps": [
            "Check LLM backend connectivity.",
            "Manually review collected evidence.",
        ],
        "confidence": 0.0,
        "error": last_error,
    }


def query_case(data):
    """
    Answers a specific user question concerning case records using contextual prompt injection.

    Parameters:
    - data (dict): Contains case context text, LLM credentials, and the query/question string in 'data'.
    """
    case_name = data.get("case_name", "Unknown")
    context = data.get("context", "")
    question = data.get("data", "")
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

    api_url = llm_config.get("url", "http://localhost:11434/api/generate")
    api_key = llm_config.get("api_key", "")
    timeout = llm_config.get("timeout", 120)

    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    system_prompt = (
        "You are SPECTRE, an expert intelligence analyst. "
        "Use the provided case context to answer the user's specific question. "
        "Be concise, professional, and focus on the evidence. "
        "If the information is not in the context, state that clearly."
    )

    full_prompt = f"{system_prompt}\n\nCASE CONTEXT:\n{context}\n\nQUESTION: {question}"

    payload = {"model": model, "prompt": full_prompt, "stream": False}

    try:
        resp = session.post(api_url, json=payload, headers=headers, timeout=timeout)
        resp.raise_for_status()

        response_json = resp.json()
        raw_response = response_json.get("response", "")
        if not raw_response and "choices" in response_json:
            raw_response = response_json["choices"][0]["message"]["content"]

        return {"answer": raw_response}

    except Exception as e:
        return {"answer": f"Error querying LLM: {str(e)}"}


def analyze_image(data):
    """
    Performs visual analysis on a base64 encoded screenshot image using a vision-capable LLM (e.g. LLaVA).

    Parameters:
    - data (dict): Contains the prompt instructions ('data') and base64 string ('context').
    """
    model = data.get("model", "llava")
    prompt = data.get(
        "data",
        "Describe this image in detail. Focus on text, logos, or identifying features.",
    )
    image_base64 = data.get("context", "")
    llm_config = data.get("llm_config", {})

    api_url = llm_config.get("url", "http://localhost:11434/api/generate")
    api_key = llm_config.get("api_key", "")
    timeout = llm_config.get("timeout", 120)

    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    # Pack base64 image data inside the 'images' array field matching the Ollama vision specification.
    payload = {
        "model": model,
        "prompt": prompt,
        "images": [image_base64],
        "stream": False,
    }

    try:
        resp = session.post(api_url, json=payload, headers=headers, timeout=timeout)
        resp.raise_for_status()

        response_json = resp.json()
        raw_response = response_json.get("response", "")
        if not raw_response and "choices" in response_json:
            raw_response = response_json["choices"][0]["message"]["content"]

        return {"answer": raw_response}

    except Exception as e:
        return {"answer": f"Error performing visual analysis: {str(e)}"}


def generate_dorks(data):
    """
    Generates specialized Google Search Dorks for a target domain using LLM generation.
    Forces JSON-only output format.

    Parameters:
    - data (dict): Target domain string stored in 'data'.
    """
    target = data.get("data", "example.com")
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

    api_url = llm_config.get("url", "http://localhost:11434/api/generate")
    api_key = llm_config.get("api_key", "")
    timeout = llm_config.get("timeout", 120)

    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    system_prompt = (
        "You are a SPECTRE OSINT Specialist. "
        "Generate a list of 10 effective Google Dorks for the given target domain "
        "to find leaked documents, exposed .env files, git repositories, or sensitive logs. "
        "Your output must be a JSON object with a 'dorks' array of strings."
    )

    full_prompt = f"{system_prompt}\n\nTARGET: {target}"

    # Request JSON formatting constraint natively from the Ollama runner.
    payload = {"model": model, "prompt": full_prompt, "stream": False, "format": "json"}

    try:
        resp = session.post(api_url, json=payload, headers=headers, timeout=timeout)
        resp.raise_for_status()

        response_json = resp.json()
        raw_response = response_json.get("response", "")

        # Verify and extract the JSON output.
        extracted = extract_json(raw_response)
        if extracted:
            return extracted

        # Hand-crafted fallback splitter if JSON parsing failed.
        return {"dorks": [d.strip() for d in raw_response.split("\n") if ":" in d]}

    except Exception as e:
        return {"error": str(e)}
