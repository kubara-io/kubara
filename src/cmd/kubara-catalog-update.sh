#!/usr/bin/env bash
# Refresh the official catalog pins and regenerate a GitOps working tree.
set -euo pipefail

for tool in yq jq oras kubara; do
    command -v "$tool" >/dev/null || { echo "Missing command: $tool" >&2; exit 1; }
done

cd "${1:-.}"
config=${2:-config.yaml}
tmp=$(mktemp -d)
restore=false
cleanup() {
    if [ "$restore" = true ]; then
        cp "$tmp/original.yaml" "$config"
    fi
    rm -rf "$tmp"
}
trap cleanup EXIT
cp "$config" "$tmp/original.yaml"
cp "$config" "$tmp/updated.yaml"

# Require explicit pins so generation cannot silently use implicit CLI defaults.
yq -o=json '.' "$config" | jq -e '
    (.bootstrapCatalog | type == "string") and
    (.clusters | type == "array" and length > 0) and
    all(.clusters[]; .catalogs | type == "array" and length > 0 and all(.[]; type == "string"))
' >/dev/null
references=$(yq -r '.bootstrapCatalog, .clusters[].catalogs[]' "$config" | sort -u)
version_pattern='^(0|[1-9][0-9]*)[.](0|[1-9][0-9]*)[.](0|[1-9][0-9]*)$'

for catalog in bootstrap general; do
    repository="ghcr.io/kubara-io/catalogs/$catalog"
    tags=$(oras repo tags "$repository")
    latest=$(printf '%s\n' "$tags" | awk -v pattern="$version_pattern" '$0 ~ pattern' |
        sort -t . -k1,1n -k2,2n -k3,3n | tail -n 1)
    [ -n "$latest" ] || { echo "No stable catalog tags: $repository" >&2; exit 1; }
    found=false
    while IFS= read -r reference; do
        [[ "$reference" == "oci://$repository:"* ]] || continue
        found=true
        current=${reference##*:}
        [[ "$current" =~ $version_pattern ]] || { echo "Expected a stable catalog pin: $reference" >&2; exit 1; }
        newest=$(printf '%s\n%s\n' "$current" "$latest" | sort -t . -k1,1n -k2,2n -k3,3n | tail -n 1)
        updated="oci://$repository:$newest"
        kubara catalog pull "$updated"
        if [ "$current" != "$newest" ]; then
            OLD_REFERENCE="$reference" NEW_REFERENCE="$updated" yq -i '
                (.bootstrapCatalog, .clusters[].catalogs[]) |=
                    (select(. == strenv(OLD_REFERENCE)) = strenv(NEW_REFERENCE))
            ' "$tmp/updated.yaml"
        fi
    done <<< "$references"
    [ "$found" = true ] || { echo "Missing explicit catalog pin: $repository" >&2; exit 1; }
done

restore=true
cp "$tmp/updated.yaml" "$config"
kubara --config-file "$config" generate
restore=false
