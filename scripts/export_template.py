"""Export application source and committed assets from the current working tree."""
import argparse
from pathlib import Path
import shutil
import subprocess


ROOT = Path(__file__).resolve().parents[1]
TOP_FILES = {
    ".dockerignore", ".env.example", ".gitignore", ".gitattributes", "LICENSE",
    "NOTICE", "THIRD-PARTY-LICENSES.md", "VERSION", "README.md", "AGENTS.md",
    "go.mod", "go.sum", "Dockerfile", "Dockerfile.dev", "docker-compose.dev.yml", "docker-compose.yml",
}
GO_PACKAGES = {"common", "constant", "controller", "dto", "i18n", "logger",
               "middleware", "model", "oauth", "pkg", "router", "service", "setting", "types"}
WEB_TOP = {"package.json","bun.lock","index.html","rsbuild.config.ts","vitest.config.ts",
           "tsconfig.json","tsconfig.app.json","tsconfig.node.json",".oxlintrc.json",
           ".oxfmtrc.json","components.json",".node-version","AGENTS.md",".gitignore",".env.example","knip.config.ts"}
DOCS = {"docs/PRODUCT.md","docs/ARCHITECTURE.md","docs/authentication.md","docs/TASKS.md","docs/session-handoff.md","docs/CHANGELOG.md","docs/translation-glossary.md"}
SOURCE_SUFFIXES = {".go",".ts",".tsx",".mjs",".css",".py"}
ASSET_SUFFIXES = {".json",".yaml",".yml",".lua",".svg",".png",".ico",".jpg",".jpeg",".webp",".gif",".woff",".woff2"}


def selected(p, tracked):
    parts = p.parts
    name = p.as_posix()
    if name in TOP_FILES or name in DOCS:
        return True
    if parts[:2] == ("docs", "images"):
        return len(parts) == 3 and p.name in {"overview.png", "tickets.png", "ticket-detail.png", "audit.png", "settings.png"}
    if parts[:2] == (".agents", "skills"):
        return tracked and p.suffix in {".md", ".txt"}
    if len(parts) == 1:
        return p.suffix == ".go"
    if parts[0] in GO_PACKAGES:
        return p.suffix == ".go" or (tracked and p.suffix in ASSET_SUFFIXES)
    if parts[0] == "examples" and parts[1] == "tickets":
        return p.suffix in {".go",".py"}
    if parts[0] == "scripts":
        return len(parts)==2 and p.suffix==".py"
    if parts[0] == "web":
        if len(parts)==2:
            return parts[1] in WEB_TOP
        if parts[1] in ("src","scripts","public"):
            return p.suffix in SOURCE_SUFFIXES or (tracked and p.suffix in ASSET_SUFFIXES)
    return False


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("destination",type=Path)
    parser.add_argument("--tracked-only", action="store_true", help="include only indexed source paths")
    args=parser.parse_args()
    dest=args.destination.resolve()
    if dest.is_relative_to(ROOT):
        parser.error("Choose a new directory outside the source tree.")
    def paths(*args):
        output=subprocess.check_output(["git","ls-files","-z",*args],cwd=ROOT)
        return {Path(item.decode("utf-8")) for item in output.split(b"\0") if item}
    tracked=paths("--cached")
    candidates=tracked if args.tracked_only else tracked | paths("--others","--exclude-standard")
    dest.mkdir(parents=True,exist_ok=False)
    count=0
    for rel in sorted(candidates):
        if not selected(rel,rel in tracked):
            continue
        source=ROOT/rel
        if not source.is_file() or source.is_symlink() or not source.resolve().is_relative_to(ROOT):
            continue
        target=dest/rel
        target.parent.mkdir(parents=True,exist_ok=True)
        shutil.copyfile(source,target)
        count+=1
    print(f"Exported {count} source and asset files to {dest}")


if __name__=="__main__":
    main()
