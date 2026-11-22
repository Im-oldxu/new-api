"""Remove the shipped ticket sample's source from a template copy."""
import argparse
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
FILES=(
    "web/src/features/tickets/ticket-status-badge.tsx",
    "examples/tickets/overview.go", "web/src/features/tickets/overview.tsx", "web/src/features/tickets/settings.tsx",
    "web/src/features/tickets/__tests__/conversation.test.tsx",
    "web/src/features/tickets/__tests__/overview.test.tsx",
    "web/src/features/tickets/conversation.tsx",
    "web/src/features/tickets/__tests__/editor.test.tsx",
    "tickets_example.go","examples/tickets/tickets.go","examples/tickets/tickets_test.go",
    "examples/tickets/demo.py","web/src/routes/_authenticated/tickets/index.tsx",
    "web/src/features/tickets/navigation.ts","web/src/features/tickets/api.ts","web/src/features/tickets/editor.tsx","web/src/features/tickets/index.tsx",
)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apply",action="store_true",help="apply the displayed source changes")
    args=parser.parse_args()
    navigation=ROOT/"web/src/business-navigation.ts"
    overview=ROOT/"web/src/business-overview.ts"
    files=[ROOT/f for f in FILES if (ROOT/f).is_file()]
    for p in [*files,navigation,overview]:
        if p.is_symlink() or not p.resolve().is_relative_to(ROOT):
            parser.error("Unexpected source path.")
    text=navigation.read_text(encoding="utf-8")
    updated = "".join(line for line in text.splitlines(keepends=True)
                      if not (line.strip() == "ticketNavigation," or
                              line.strip() == "import { ticketNavigation } from '@/features/tickets/navigation'"))
    if "ticketNavigation" in updated:
        parser.error("Custom ticket navigation detected. Remove its registration manually.")
    print("Remove the ticket navigation entry and these source files:")
    for p in files:
        print(p.relative_to(ROOT))
    overview_source=overview.read_text(encoding="utf-8")
    overview_updated="".join(line for line in overview_source.splitlines(keepends=True)
                             if line.strip() != "{ id: 'tickets', component: lazy(() => import('@/features/tickets/overview')) },")
    if "features/tickets" in overview_updated:
        parser.error("Remove custom ticket overview registration manually.")
    if "lazy(" not in overview_updated:
        overview_updated=overview_updated.replace("import { lazy, type ComponentType }", "import type { ComponentType }")
    if args.apply:
        overview.write_text(overview_updated,encoding="utf-8")
        for p in files:
            p.unlink()
        navigation.write_text(updated,encoding="utf-8")
        print("Run bun run build in web to regenerate the route tree, then rebuild the Go server.")


if __name__=="__main__":
    main()
