import json
import sys
import argparse
from .llm import analyze_case, chat, query_case, analyze_image, generate_dorks
from .graph_viz import generate_visual_report
from .vector_store import index_evidence, search_evidence


def main():
    parser = argparse.ArgumentParser(description="SPECTRE Analyzer (Python)")
    parser.add_argument(
        "--task",
        help="The task to perform",
        choices=[
            "synthesize",
            "visualize",
            "chat",
            "query",
            "vision",
            "index_evidence",
            "search_evidence",
            "generate_dorks",
        ],
        required=True,
    )
    parser.add_argument("--input", help="JSON input data", required=True)

    args = parser.parse_args()

    try:
        input_data = json.loads(args.input)

        if args.task == "synthesize":
            result = analyze_case(input_data)
            print(json.dumps(result))
        elif args.task == "query":
            result = query_case(input_data)
            print(json.dumps(result))
        elif args.task == "vision":
            result = analyze_image(input_data)
            print(json.dumps(result))
        elif args.task == "chat":
            result = chat(input_data)
            print(json.dumps(result))
        elif args.task == "visualize":
            # Extract the actual graph data payload
            graph_data = input_data.get("data", {})
            result = generate_visual_report(graph_data)
            print(json.dumps(result))
        elif args.task == "index_evidence":
            case_id = input_data.get("case_id")
            files = input_data.get("files", [])
            result = index_evidence(case_id, files)
            print(json.dumps(result))
        elif args.task == "search_evidence":
            case_id = input_data.get("case_id")
            query = input_data.get("query")
            n_results = input_data.get("n_results", 3)
            result = search_evidence(case_id, query, n_results)
            print(json.dumps(result))
        elif args.task == "generate_dorks":
            result = generate_dorks(input_data)
            print(json.dumps(result))

    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
