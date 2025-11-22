#!/usr/bin/env python3
"""
Tiny bridge to run the Python POML SDK and emit JSON for a given format.
Usage: python tools/py_bridge.py --format openai_chat --file path/to/input.poml
"""

import argparse
import json
import sys
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--format", required=True, help="message_dict|dict|openai_chat|langchain|pydantic")
    parser.add_argument("--file", required=True, help="POML file to parse")
    args = parser.parse_args()

    try:
        import poml  # type: ignore
    except ImportError:
        print("ERROR: python poml package not installed. pip install poml", file=sys.stderr)
        return 1

    path = Path(args.file)
    if not path.exists():
        print(f"ERROR: file not found: {path}", file=sys.stderr)
        return 1

    body = path.read_text(encoding="utf-8")
    out = poml.poml(body, format=args.format)

    # Pydantic models are not directly serializable; convert to dict.
    try:
        import pydantic  # type: ignore

        def default(o):
            if hasattr(o, "model_dump"):
                return o.model_dump()
            if hasattr(o, "__dict__"):
                return o.__dict__
            return str(o)

        print(json.dumps(out, default=default))
    except Exception:
        print(json.dumps(out, default=str))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
