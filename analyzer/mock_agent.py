import json
import sys
import argparse


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--task")
    parser.add_argument("--input")
    args = parser.parse_args()

    input_data = json.loads(args.input)
    messages = input_data.get("messages", [])

    # Check if we just received a tool result
    if messages and messages[-1]["role"] == "system":
        last_content = messages[-1]["content"]
        if "Observation from 'collect'" in last_content:
            print(
                json.dumps(
                    {
                        "role": "assistant",
                        "content": "I have finished the DNS collection for google.com. It was successful.",
                    }
                )
            )
            return
        elif "Observation from 'read_evidence'" in last_content:
            flag = "unknown"
            if "SPECTRE_FLAG" in last_content:
                start = last_content.find("SPECTRE_FLAG")
                flag = last_content[start:].split()[0]
                flag = flag.rstrip(".")
            print(
                json.dumps(
                    {
                        "role": "assistant",
                        "content": f"I have read the WHOIS evidence and found the flag: {flag}",
                    }
                )
            )
            return

    # Simple logic to simulate an agent loop
    last_user_msg = ""
    for m in reversed(messages):
        if m["role"] == "user":
            last_user_msg = m["content"].lower()
            break

    if "run dns" in last_user_msg:
        # Simulate tool use
        print(
            json.dumps(
                {
                    "role": "assistant",
                    "tool_use": {
                        "name": "collect",
                        "arguments": {"collector": "dns", "target": "google.com"},
                    },
                }
            )
        )
    elif "search whois flag" in last_user_msg:
        # Simulate tool use for reading evidence
        print(
            json.dumps(
                {
                    "role": "assistant",
                    "tool_use": {
                        "name": "read_evidence",
                        "arguments": {"filename": "whois_flag.txt"},
                    },
                }
            )
        )
    else:
        # Default response
        print(
            json.dumps(
                {
                    "role": "assistant",
                    "content": "Hello! I am the SPECTRE mock agent. How can I help?",
                }
            )
        )


if __name__ == "__main__":
    main()
