"""Package an exported application tree and write its SHA-256 manifest."""
import argparse
import hashlib
from pathlib import Path
import zipfile

from release_notes import release_body


def package(source, destination, tag):
    release_body(tag, (source / "VERSION").read_text("utf-8"),
                 (source / "docs/CHANGELOG.md").read_text("utf-8"))
    for name in ("LICENSE", "NOTICE", "THIRD-PARTY-LICENSES.md", "README.md",
                 "go.mod", "go.sum", "main.go", "web/package.json", "web/bun.lock",
                 "tickets_example.go", "scripts/remove_ticket_example.py", ".env.example"):
        if not (source / name).is_file():
            raise ValueError(f"Missing application file: {name}")
    if destination.resolve().is_relative_to(source.resolve()):
        raise ValueError("Choose an output directory outside the exported tree")
    destination.mkdir(parents=True, exist_ok=False)
    archive = destination / f"general-admin-base-{tag}.zip"
    with zipfile.ZipFile(archive, "x") as bundle:
        for path in sorted(source.rglob("*"), key=lambda p: p.relative_to(source).as_posix()):
            if path.is_file():
                # Fixed timestamp and permissions make retries independent of checkout time.
                entry = zipfile.ZipInfo(path.relative_to(source).as_posix())
                entry.create_system = 3
                entry.external_attr = 0o100644 << 16
                bundle.writestr(entry, path.read_bytes(), compress_type=zipfile.ZIP_DEFLATED)
    with zipfile.ZipFile(archive) as bundle:
        if bundle.testzip():
            raise ValueError("Archive integrity check failed")
    with archive.open("rb") as stream:
        digest = hashlib.file_digest(stream, "sha256").hexdigest()
    (destination / "SHA256SUMS").write_text(f"{digest}  {archive.name}\n", encoding="ascii", newline="\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("destination", type=Path)
    parser.add_argument("tag")
    args = parser.parse_args()
    package(args.source, args.destination, args.tag)
