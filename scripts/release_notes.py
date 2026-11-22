"""Extract the Markdown release body for a stable tag from the project changelog."""
import argparse
from datetime import date
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
NUMBER = r"(?:0|[1-9][0-9]*)"
VERSION = rf"{NUMBER}\.{NUMBER}\.{NUMBER}"
HEADING = re.compile(rf"## \[({VERSION})\] - (\d{{4}}-\d{{2}}-\d{{2}})")


def release_body(tag, version_file, changelog):
    if not re.fullmatch(rf"v{VERSION}", tag) or version_file.strip() != tag:
        raise ValueError("Expected a stable vX.Y.Z tag matching VERSION")

    sections = {}
    current = None
    fence = ""
    for line in changelog.splitlines():
        marker = re.match(r"^ {0,3}(`{3,}|~{3,})(.*)$", line)
        if fence:
            if marker and marker[1].startswith(fence) and not marker[2].strip():
                fence = ""
        elif marker:
            fence = marker[1]
        elif re.match(r"^##(?:\s|$)", line):
            heading = line.rstrip()
            if heading == "## [Unreleased]" and not sections:
                current = "Unreleased"
            else:
                match = HEADING.fullmatch(heading)
                if not match or "Unreleased" not in sections:
                    raise ValueError(f"Invalid changelog heading: {line}")
                current = match[1]
                date.fromisoformat(match[2])
            if current in sections:
                raise ValueError(f"Duplicate changelog heading: {line}")
            sections[current] = []
            continue
        if current is not None:
            sections[current].append(line)
    lines = sections.get(tag[1:], [])
    body = "\n".join(lines).strip("\n")
    visible = re.sub(r"<!--.*?-->", "", body, flags=re.S)
    if not any(line.strip() and not line.lstrip().startswith("#") for line in visible.splitlines()):
        raise ValueError(f"Missing or empty release notes for {tag}")
    return body + "\n"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag")
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    try:
        body = release_body(args.tag, (ROOT / "VERSION").read_text("utf-8"),
                            (ROOT / "docs/CHANGELOG.md").read_text("utf-8"))
        args.output.write_text(body, encoding="utf-8", newline="\n")
    except (ValueError, OSError) as error:
        parser.exit(1, f"{error}\n")
