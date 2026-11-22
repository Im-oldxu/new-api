#!/usr/bin/env bash
set -euo pipefail

tag=$GITHUB_REF_NAME
commit=$(git rev-parse HEAD)
assets=$RUNNER_TEMP/assets
notes=$RUNNER_TEMP/notes.md
bundle="general-admin-base-$tag.zip"
python3 scripts/release_notes.py "$tag" "$notes"
(cd "$assets" && sha256sum -c SHA256SUMS)
test "$(sha256sum "$assets/$bundle" | cut -d' ' -f1)  $bundle" = "$(cat "$assets/SHA256SUMS")"

verify_tag() {
  local object kind sha
  object=$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$tag" --jq '.object')
  kind=$(jq -r .type <<< "$object")
  sha=$(jq -r .sha <<< "$object")
  while [[ "$kind" == tag ]]; do
    object=$(gh api "repos/$GITHUB_REPOSITORY/git/tags/$sha" --jq '.object')
    kind=$(jq -r .type <<< "$object")
    sha=$(jq -r .sha <<< "$object")
  done
  [[ "$kind" == commit && "$sha" == "$commit" ]]
}

verify_tag
releases=$(gh api --paginate "repos/$GITHUB_REPOSITORY/releases?per_page=100" |
  jq -sc --arg tag "$tag" 'add // [] | map(select(.tag_name == $tag))')
case $(jq length <<< "$releases") in
  0) gh release create "$tag" --draft --verify-tag --target "$commit" --title "$tag" --notes-file "$notes" ;;
  1) ;;
  *) echo "Multiple releases for $tag" >&2; exit 1 ;;
esac

release=$(gh release view "$tag" --json isDraft,name,body,assets)
test "$(jq -r .name <<< "$release")" = "$tag"
test "$(jq -r .body <<< "$release" | tr -d '\r')" = "$(cat "$notes")"
jq -e --arg zip "$bundle" 'all(.assets[]; .name == $zip or .name == "SHA256SUMS")' <<< "$release" >/dev/null
draft=$(jq -r .isDraft <<< "$release")
for file in "$bundle" SHA256SUMS; do
  if ! jq -e --arg name "$file" 'any(.assets[]; .name == $name)' <<< "$release" >/dev/null; then
    test "$draft" = true
    gh release upload "$tag" "$assets/$file"
  fi
done
download=$(mktemp -d)
gh release download "$tag" --dir "$download" --pattern "$bundle" --pattern SHA256SUMS
cmp "$assets/SHA256SUMS" "$download/SHA256SUMS"
cmp "$assets/$bundle" "$download/$bundle"
verify_tag
if [[ "$draft" == true ]]; then
  gh release edit "$tag" --draft=false
fi
