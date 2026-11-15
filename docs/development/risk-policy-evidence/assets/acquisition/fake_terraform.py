#!/usr/bin/env python3
"""Count subprocesses and model an immutable saved plan with synthetic JSON bytes."""

import json
import os
from pathlib import Path
import sys


def main():
    args = sys.argv[1:]
    with Path(os.environ["ACQUISITION_CALL_LOG"]).open("a") as log:
        log.write(json.dumps({"args": args, "cwd": os.getcwd()}) + "\n")
    if args == ["version", "-json"]:
        print(json.dumps({"terraform_version": "1.14.0", "platform": "linux_amd64", "provider_selections": {}}))
    elif args in (["workspace", "show"], ["workspace", "show", "-no-color"]):
        print("default")
    elif args and args[0] == "plan":
        outputs = [arg.removeprefix("-out=") for arg in args if arg.startswith("-out=")]
        if len(outputs) != 1:
            raise ValueError("experiment requires exactly one saved artifact")
        Path(outputs[0]).write_bytes(Path(os.environ["ACQUISITION_PLAN_FIXTURE"]).read_bytes())
    elif len(args) >= 3 and args[:2] == ["show", "-json"]:
        sys.stdout.buffer.write(Path(args[-1]).read_bytes())
    else:
        raise ValueError(f"unexpected Terraform invocation: {args!r}")


if __name__ == "__main__":
    main()
