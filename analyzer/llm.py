import requests
import json
import re
import time
import sys
import copy
from typing import Dict, Any, List, Optional, Tuple

# Global HTTP session object to enable connection pooling across inferences.
session = requests.Session()


def extract_json(text: str) -> Optional[Any]:
    """
    Robustly extracts valid JSON from a string, handling markdown code fences,
    conversational preamble, or postamble text from any LLM.
    """
    if not text:
        return None
    text = text.strip()

    # 1. Direct parse attempt
    try:
        return json.loads(text)
    except Exception:
        pass

    # 2. Look for markdown json code fence: ```json ... ```
    fence_match = re.search(r"```(?:json)?\s*([\s\S]*?)\s*```", text, re.IGNORECASE)
    if fence_match:
        try:
            return json.loads(fence_match.group(1).strip())
        except Exception:
            pass

    # 3. Search for outermost matching curly brackets { ... }
    first_brace = text.find("{")
    last_brace = text.rfind("}")
    if first_brace != -1 and last_brace != -1 and last_brace > first_brace:
        candidate = text[first_brace : last_brace + 1]
        try:
            return json.loads(candidate)
        except Exception:
            pass

    # 4. Search for outermost square brackets [ ... ]
    first_bracket = text.find("[")
    last_bracket = text.rfind("]")
    if first_bracket != -1 and last_bracket != -1 and last_bracket > first_bracket:
        candidate = text[first_bracket : last_bracket + 1]
        try:
            return json.loads(candidate)
        except Exception:
            pass

    # 5. Regex search as a last resort
    match = re.search(r"\{[\s\S]*\}", text)
    if match:
        try:
            return json.loads(match.group())
        except Exception:
            pass

    return None


def detect_provider(llm_config: Dict[str, Any]) -> str:
    """
    Infers the LLM provider from the config provider field or URL pattern.
    """
    provider = (llm_config.get("provider") or "").lower().strip()
    if provider and provider != "auto":
        return provider

    url = (llm_config.get("url") or "").lower()
    if "anthropic.com" in url:
        return "anthropic"
    if "googleapis.com" in url or "gemini" in url:
        return "gemini"
    if "api.groq.com" in url:
        return "groq"
    if "openrouter.ai" in url:
        return "openrouter"
    if "1234" in url:  # Default LM Studio port
        return "lmstudio"
    if "api.openai.com" in url or "/v1" in url:
        return "openai"
    if "11434" in url or "ollama" in url:
        return "ollama"

    return "ollama"


class BaseLLMClient:
    """Base interface for model provider clients."""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.api_key = config.get("api_key", "")
        self.timeout = config.get("timeout", 120)
        self.raw_url = config.get("url", "")

    def chat(self, messages: List[Dict[str, Any]], tools: List[Any], model: str) -> Dict[str, Any]:
        raise NotImplementedError

    def generate(self, prompt: str, model: str, json_mode: bool = False) -> str:
        raise NotImplementedError

    def vision(self, prompt: str, image_base64: str, model: str) -> str:
        raise NotImplementedError


class OllamaClient(BaseLLMClient):
    """Client for local Ollama instances."""

    def _get_base_url(self) -> str:
        url = self.raw_url or "http://localhost:11434"
        url = re.sub(r"/api/(?:chat|generate|tags)?$", "", url.rstrip("/"))
        return url or "http://localhost:11434"

    def _get_headers(self) -> Dict[str, str]:
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers

    def chat(self, messages: List[Dict[str, Any]], tools: List[Any], model: str) -> Dict[str, Any]:
        # If raw_url ends in /api/generate from older configs, replace with /api/chat
        if self.raw_url and "/api/chat" in self.raw_url:
            endpoint = self.raw_url
        elif self.raw_url and "/api/generate" in self.raw_url:
            endpoint = self.raw_url.replace("/api/generate", "/api/chat")
        else:
            endpoint = f"{self._get_base_url()}/api/chat"

        payload = {"model": model, "messages": messages, "stream": False}
        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()

        content = ""
        if "message" in data:
            content = data["message"].get("content", "")
        elif "choices" in data:
            content = data["choices"][0]["message"].get("content", "")
        elif "response" in data:
            content = data["response"]

        # Parse output for tool use JSON
        tool_call = extract_json(content)
        if tool_call and "tool_use" in tool_call:
            return {"role": "assistant", "tool_use": tool_call["tool_use"]}

        return {"role": "assistant", "content": content}

    def generate(self, prompt: str, model: str, json_mode: bool = False) -> str:
        if self.raw_url and "/api/generate" in self.raw_url:
            endpoint = self.raw_url
        elif self.raw_url and "/api/chat" in self.raw_url:
            endpoint = self.raw_url.replace("/api/chat", "/api/generate")
        else:
            endpoint = f"{self._get_base_url()}/api/generate"

        payload: Dict[str, Any] = {"model": model, "prompt": prompt, "stream": False}
        if json_mode:
            payload["format"] = "json"

        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        return data.get("response", "") or (data.get("choices", [{}])[0].get("message", {}).get("content", ""))

    def vision(self, prompt: str, image_base64: str, model: str) -> str:
        if self.raw_url and "/api/generate" in self.raw_url:
            endpoint = self.raw_url
        elif self.raw_url and "/api/chat" in self.raw_url:
            endpoint = self.raw_url.replace("/api/chat", "/api/generate")
        else:
            endpoint = f"{self._get_base_url()}/api/generate"

        payload = {
            "model": model,
            "prompt": prompt,
            "images": [image_base64],
            "stream": False,
        }
        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        return data.get("response", "") or (data.get("choices", [{}])[0].get("message", {}).get("content", ""))


class OpenAICompatibleClient(BaseLLMClient):
    """
    Universal client for OpenAI and any OpenAI-compatible provider:
    OpenAI, Groq, OpenRouter, DeepSeek, Together, LM Studio, LocalAI, vLLM, etc.
    """

    def __init__(self, config: Dict[str, Any], default_base: str = "https://api.openai.com/v1"):
        super().__init__(config)
        self.default_base = default_base

    def _get_base_url(self) -> str:
        url = (self.raw_url or self.default_base).rstrip("/")
        url = re.sub(r"/chat/completions$", "", url)
        url = re.sub(r"/completions$", "", url)
        return url or self.default_base

    def _get_headers(self) -> Dict[str, str]:
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers

    def chat(self, messages: List[Dict[str, Any]], tools: List[Any], model: str) -> Dict[str, Any]:
        endpoint = f"{self._get_base_url()}/chat/completions"
        payload: Dict[str, Any] = {
            "model": model,
            "messages": messages,
            "stream": False,
        }

        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()

        choice = data.get("choices", [{}])[0].get("message", {})
        content = choice.get("content", "") or ""

        # Check native OpenAI tool_calls
        if "tool_calls" in choice and choice["tool_calls"]:
            tc = choice["tool_calls"][0]
            fn = tc.get("function", {})
            fn_name = fn.get("name", "")
            raw_args = fn.get("arguments", "{}")
            try:
                fn_args = json.loads(raw_args)
            except Exception:
                fn_args = {"raw": raw_args}
            return {"role": "assistant", "tool_use": {"name": fn_name, "arguments": fn_args}}

        # Parse output for prompt-based tool use JSON
        tool_call = extract_json(content)
        if tool_call and isinstance(tool_call, dict) and "tool_use" in tool_call:
            return {"role": "assistant", "tool_use": tool_call["tool_use"]}

        return {"role": "assistant", "content": content}

    def generate(self, prompt: str, model: str, json_mode: bool = False) -> str:
        endpoint = f"{self._get_base_url()}/chat/completions"
        messages = [{"role": "user", "content": prompt}]
        payload: Dict[str, Any] = {
            "model": model,
            "messages": messages,
            "stream": False,
        }

        if json_mode:
            payload["response_format"] = {"type": "json_object"}

        try:
            resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
            resp.raise_for_status()
        except requests.HTTPError as err:
            # If provider doesn't support response_format={"type": "json_object"}, retry without it
            if json_mode and resp.status_code in (400, 422):
                payload.pop("response_format", None)
                resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
                resp.raise_for_status()
            else:
                raise err

        data = resp.json()
        return data.get("choices", [{}])[0].get("message", {}).get("content", "") or ""

    def vision(self, prompt: str, image_base64: str, model: str) -> str:
        endpoint = f"{self._get_base_url()}/chat/completions"
        image_url = f"data:image/jpeg;base64,{image_base64}"
        messages = [
            {
                "role": "user",
                "content": [
                    {"type": "text", "text": prompt},
                    {"type": "image_url", "image_url": {"url": image_url}},
                ],
            }
        ]
        payload = {
            "model": model,
            "messages": messages,
            "stream": False,
        }
        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        return data.get("choices", [{}])[0].get("message", {}).get("content", "") or ""


class AnthropicClient(BaseLLMClient):
    """Client for Anthropic Claude API."""

    def _get_base_url(self) -> str:
        url = (self.raw_url or "https://api.anthropic.com").rstrip("/")
        url = re.sub(r"/v1(/messages)?$", "", url)
        url = re.sub(r"/messages$", "", url)
        return url or "https://api.anthropic.com"

    def _get_headers(self) -> Dict[str, str]:
        return {
            "Content-Type": "application/json",
            "x-api-key": self.api_key,
            "anthropic-version": "2023-06-01",
        }

    def _format_messages(self, messages: List[Dict[str, Any]]) -> Tuple[Optional[str], List[Dict[str, Any]]]:
        system_prompt = None
        formatted = []
        for m in messages:
            role = m.get("role")
            content = m.get("content", "")
            if role == "system":
                if system_prompt is None:
                    system_prompt = content
                else:
                    system_prompt += "\n\n" + content
            else:
                formatted.append({"role": role if role in ("user", "assistant") else "user", "content": content})

        if not formatted:
            formatted = [{"role": "user", "content": "Hello"}]
        return system_prompt, formatted

    def chat(self, messages: List[Dict[str, Any]], tools: List[Any], model: str) -> Dict[str, Any]:
        endpoint = f"{self._get_base_url()}/v1/messages"
        system_prompt, formatted_msgs = self._format_messages(messages)

        payload: Dict[str, Any] = {
            "model": model,
            "messages": formatted_msgs,
            "max_tokens": self.config.get("max_tokens", 4096),
        }
        if system_prompt:
            payload["system"] = system_prompt

        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()

        content = ""
        for block in data.get("content", []):
            if block.get("type") == "text":
                content += block.get("text", "")
            elif block.get("type") == "tool_use":
                return {
                    "role": "assistant",
                    "tool_use": {
                        "name": block.get("name", ""),
                        "arguments": block.get("input", {}),
                    },
                }

        tool_call = extract_json(content)
        if tool_call and isinstance(tool_call, dict) and "tool_use" in tool_call:
            return {"role": "assistant", "tool_use": tool_call["tool_use"]}

        return {"role": "assistant", "content": content}

    def generate(self, prompt: str, model: str, json_mode: bool = False) -> str:
        endpoint = f"{self._get_base_url()}/v1/messages"
        payload = {
            "model": model,
            "messages": [{"role": "user", "content": prompt}],
            "max_tokens": self.config.get("max_tokens", 4096),
        }
        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        content = ""
        for block in data.get("content", []):
            if block.get("type") == "text":
                content += block.get("text", "")
        return content

    def vision(self, prompt: str, image_base64: str, model: str) -> str:
        endpoint = f"{self._get_base_url()}/v1/messages"
        payload = {
            "model": model,
            "max_tokens": self.config.get("max_tokens", 4096),
            "messages": [
                {
                    "role": "user",
                    "content": [
                        {
                            "type": "image",
                            "source": {
                                "type": "base64",
                                "media_type": "image/jpeg",
                                "data": image_base64,
                            },
                        },
                        {"type": "text", "text": prompt},
                    ],
                }
            ],
        }
        resp = session.post(endpoint, json=payload, headers=self._get_headers(), timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        content = ""
        for block in data.get("content", []):
            if block.get("type") == "text":
                content += block.get("text", "")
        return content


class GeminiClient(BaseLLMClient):
    """
    Client for Google Gemini.
    Supports either Gemini OpenAI-compatible endpoint or native generateContent API.
    """

    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.openai_fallback = OpenAICompatibleClient(
            config, default_base="https://generativelanguage.googleapis.com/v1beta/openai"
        )

    def _is_openai_mode(self) -> bool:
        url = (self.raw_url or "").lower()
        return not url or "/openai" in url

    def chat(self, messages: List[Dict[str, Any]], tools: List[Any], model: str) -> Dict[str, Any]:
        if self._is_openai_mode():
            return self.openai_fallback.chat(messages, tools, model)

        # Native Gemini generateContent API
        api_key = self.api_key
        url = self.raw_url or "https://generativelanguage.googleapis.com/v1beta"
        url = re.sub(r"/models/.*$", "", url.rstrip("/"))
        endpoint = f"{url}/models/{model}:generateContent?key={api_key}"

        contents = []
        for m in messages:
            role = "model" if m.get("role") == "assistant" else "user"
            contents.append({"role": role, "parts": [{"text": m.get("content", "")}]})

        payload = {"contents": contents}
        resp = session.post(endpoint, json=payload, headers={"Content-Type": "application/json"}, timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()

        text = ""
        candidates = data.get("candidates", [])
        if candidates:
            parts = candidates[0].get("content", {}).get("parts", [])
            if parts:
                text = parts[0].get("text", "")

        tool_call = extract_json(text)
        if tool_call and isinstance(tool_call, dict) and "tool_use" in tool_call:
            return {"role": "assistant", "tool_use": tool_call["tool_use"]}

        return {"role": "assistant", "content": text}

    def generate(self, prompt: str, model: str, json_mode: bool = False) -> str:
        if self._is_openai_mode():
            return self.openai_fallback.generate(prompt, model, json_mode)

        api_key = self.api_key
        url = self.raw_url or "https://generativelanguage.googleapis.com/v1beta"
        url = re.sub(r"/models/.*$", "", url.rstrip("/"))
        endpoint = f"{url}/models/{model}:generateContent?key={api_key}"

        payload: Dict[str, Any] = {
            "contents": [{"role": "user", "parts": [{"text": prompt}]}]
        }
        if json_mode:
            payload["generationConfig"] = {"responseMimeType": "application/json"}

        resp = session.post(endpoint, json=payload, headers={"Content-Type": "application/json"}, timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        candidates = data.get("candidates", [])
        if candidates:
            parts = candidates[0].get("content", {}).get("parts", [])
            if parts:
                return parts[0].get("text", "")
        return ""

    def vision(self, prompt: str, image_base64: str, model: str) -> str:
        if self._is_openai_mode():
            return self.openai_fallback.vision(prompt, image_base64, model)

        api_key = self.api_key
        url = self.raw_url or "https://generativelanguage.googleapis.com/v1beta"
        url = re.sub(r"/models/.*$", "", url.rstrip("/"))
        endpoint = f"{url}/models/{model}:generateContent?key={api_key}"

        payload = {
            "contents": [
                {
                    "role": "user",
                    "parts": [
                        {"text": prompt},
                        {"inlineData": {"mimeType": "image/jpeg", "data": image_base64}},
                    ],
                }
            ]
        }
        resp = session.post(endpoint, json=payload, headers={"Content-Type": "application/json"}, timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        candidates = data.get("candidates", [])
        if candidates:
            parts = candidates[0].get("content", {}).get("parts", [])
            if parts:
                return parts[0].get("text", "")
        return ""


def get_llm_client(llm_config: Dict[str, Any]) -> BaseLLMClient:
    """
    Factory returning the appropriate provider client for the given configuration.
    """
    provider = detect_provider(llm_config)

    if provider == "ollama":
        return OllamaClient(llm_config)
    elif provider == "anthropic":
        return AnthropicClient(llm_config)
    elif provider == "gemini":
        return GeminiClient(llm_config)
    elif provider == "groq":
        return OpenAICompatibleClient(llm_config, default_base="https://api.groq.com/openai/v1")
    elif provider == "openrouter":
        return OpenAICompatibleClient(llm_config, default_base="https://openrouter.ai/api/v1")
    elif provider == "lmstudio":
        return OpenAICompatibleClient(llm_config, default_base="http://localhost:1234/v1")
    elif provider == "localai":
        return OpenAICompatibleClient(llm_config, default_base="http://localhost:8080/v1")
    else:
        # Default OpenAI & generic OpenAI-compatible
        return OpenAICompatibleClient(llm_config, default_base="https://api.openai.com/v1")


def chat(data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Orchestrates an interactive chat turn between the user and the SPECTRE agent.
    Supports local (Ollama, LM Studio) and cloud (OpenAI, Anthropic, Gemini, Groq, OpenRouter).
    """
    messages = copy.deepcopy(data.get("messages", []))
    tools = data.get("tools", [])
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

    # Define system instructions outlining agent persona and tool format
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

    if not messages or messages[0].get("role") != "system":
        messages.insert(0, system_msg)

    client = get_llm_client(llm_config)

    try:
        return client.chat(messages=messages, tools=tools, model=model)
    except Exception as e:
        provider = detect_provider(llm_config)
        last_user_msg = ""
        for m in reversed(messages):
            if m.get("role") == "user":
                last_user_msg = m.get("content", "").lower()
                break

        if "hello" in last_user_msg or "hi" in last_user_msg:
            return {
                "role": "assistant",
                "content": f"Hello! (Fallback Mode) I'm currently unable to reach my LLM backend ({provider}), but I can still help with basic commands.",
            }

        return {
            "role": "assistant",
            "content": f"I'm sorry, I encountered an error communicating with {provider}: {str(e)}. Please check your provider settings or server status.",
        }


def analyze_case(data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Synthesizes and consolidates gathered intelligence records into a structured findings report.
    Forces JSON-only formatting and works across all local and cloud LLMs.
    """
    case_name = data.get("case_name", "Unknown")
    context = data.get("context", "")
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

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

    full_prompt = f"{system_prompt}\n\nCASE DATA ({case_name}):\n{context}"
    client = get_llm_client(llm_config)

    retries = 3
    last_error = ""

    for attempt in range(retries):
        try:
            raw_response = client.generate(full_prompt, model=model, json_mode=True)
            extracted = extract_json(raw_response)
            if extracted and isinstance(extracted, dict):
                # Ensure expected keys exist
                extracted.setdefault("findings", [])
                extracted.setdefault("risks", [])
                extracted.setdefault("connections", [])
                extracted.setdefault("next_steps", [])
                extracted.setdefault("missing_data", [])
                extracted.setdefault("suggested_collectors", [])
                extracted.setdefault("confidence", 0.8)
                return extracted

            last_error = "No valid JSON found in LLM response"
        except Exception as e:
            last_error = str(e)

        if attempt < retries - 1:
            time.sleep(1)

    provider = detect_provider(llm_config)
    return {
        "findings": [f"LLM synthesis unavailable ({provider}). Raw data review required."],
        "risks": ["Unable to perform AI-assisted risk assessment."],
        "connections": [],
        "next_steps": [
            f"Check LLM backend connectivity for provider '{provider}'.",
            "Manually review collected evidence.",
        ],
        "confidence": 0.0,
        "error": last_error,
    }


def query_case(data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Answers a specific user question concerning case records using contextual prompt injection.
    """
    case_name = data.get("case_name", "Unknown")
    context = data.get("context", "")
    question = data.get("data", "")
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

    system_prompt = (
        "You are SPECTRE, an expert intelligence analyst. "
        "Use the provided case context to answer the user's specific question. "
        "Be concise, professional, and focus on the evidence. "
        "If the information is not in the context, state that clearly."
    )

    full_prompt = f"{system_prompt}\n\nCASE CONTEXT ({case_name}):\n{context}\n\nQUESTION: {question}"
    client = get_llm_client(llm_config)

    try:
        raw_response = client.generate(full_prompt, model=model, json_mode=False)
        return {"answer": raw_response}
    except Exception as e:
        return {"answer": f"Error querying LLM: {str(e)}"}


def analyze_image(data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Performs visual analysis on a base64 encoded screenshot image using vision-capable models.
    Supports Ollama (LLaVA), OpenAI (GPT-4o), Anthropic (Claude 3.5), Gemini, etc.
    """
    model = data.get("model", "llava")
    prompt = data.get(
        "data",
        "Describe this image in detail. Focus on text, logos, or identifying features.",
    )
    image_base64 = data.get("context", "")
    llm_config = data.get("llm_config", {})

    client = get_llm_client(llm_config)

    try:
        raw_response = client.vision(prompt, image_base64, model)
        return {"answer": raw_response}
    except Exception as e:
        return {"answer": f"Error performing visual analysis: {str(e)}"}


def generate_dorks(data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Generates specialized Google Search Dorks for a target domain using LLM generation.
    Forces JSON-only output format across all providers.
    """
    target = data.get("data", "example.com")
    model = data.get("model", "llama3")
    llm_config = data.get("llm_config", {})

    system_prompt = (
        "You are a SPECTRE OSINT Specialist. "
        "Generate a list of 10 effective Google Dorks for the given target domain "
        "to find leaked documents, exposed .env files, git repositories, or sensitive logs. "
        "Your output must be a JSON object with a 'dorks' array of strings."
    )

    full_prompt = f"{system_prompt}\n\nTARGET: {target}"
    client = get_llm_client(llm_config)

    try:
        raw_response = client.generate(full_prompt, model=model, json_mode=True)
        extracted = extract_json(raw_response)
        if extracted and isinstance(extracted, dict) and "dorks" in extracted:
            return extracted

        return {"dorks": [d.strip() for d in raw_response.split("\n") if ":" in d or "site:" in d]}
    except Exception as e:
        return {"error": str(e)}
