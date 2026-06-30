#!/usr/bin/env python3
"""Generate the site-root ``/llms.txt`` from the deployed default version.

mike publishes each docs version under ``/<version>/`` and leaves the site root
as a redirect stub, so there is no real ``/llms.txt`` at the conventional
discovery path (crawlers and agents probe the root, like ``robots.txt``). This
script writes one.

The default alias (e.g. ``latest-stable``) is a moving target — it is a symlink
that points at whatever version is current — so linking the site-root index at
``/latest-stable/...`` would hand out evergreen URLs whose content drifts and
can 404 once a page is renamed or removed. Instead this script resolves the
alias to its concrete version via ``versions.json`` and links at version
permalinks (``/<version>/...``), which stay valid. The live ``/llms.txt`` is
regenerated on every stable release, so it always advertises the newest
version.

Concretely it:

* resolves the alias to a concrete version, reads that version's per-version
  ``llms.txt`` (which has root-relative links),
* rewrites those links to absolute URLs into ``/<version>/`` so they resolve
  from the site root, and
* appends an ``## Other versions`` section listing every version's
  ``llms.txt``, read from ``versions.json``.

It is meant to run against a checked-out ``gh-pages`` worktree, after
``mike set-default``. Uses only the standard library.

Usage:
    build_root_llms.py <gh-pages-dir> [--site-url URL] [--default ALIAS]
                       [--allow-missing-default]
"""

import argparse
import json
import re
import sys
from pathlib import Path

DEFAULT_SITE_URL = "https://docs.kubara.io"
DEFAULT_ALIAS = "latest-stable"

# Matches the destination of a Markdown link whose target is *relative* (no
# scheme, leading slash or fragment). Absolute/external links are deliberately
# not matched, so they pass through untouched even if their URL contains a `)`.
# Internal targets are percent-encoded by the hook, so they never contain `)`
# or whitespace, which makes `[^)\s]+` exact for the links we do rewrite.
# The `(?<!\\)` requires an *unescaped* `]` delimiter, so an escaped bracket in
# the link label (the hook emits `\]` for a literal `]` in a nav title) is never
# mistaken for the real link delimiter.
_REL_LINK_RE = re.compile(
    r"(?<!\\)\]\((?![a-z][\w+.\-]*:|//|/|#)([^)\s]+)\)", re.IGNORECASE
)


def build_root_llms(gh_pages: Path, site_url: str, alias: str) -> str:
    site_url = site_url.rstrip("/")
    versions = _load_versions(gh_pages)
    version = _resolve_alias(versions, alias)

    src = gh_pages / version / "llms.txt"
    if not src.exists():
        # Fail closed: this runs right after `mike set-default <alias>`, so the
        # default version's llms.txt must exist. A missing file means the hook
        # did not run or mike's layout changed — surface it instead of silently
        # leaving a stale site-root index in place.
        raise FileNotFoundError(
            f"{src} not found; expected default version '{version}' (alias "
            f"'{alias}') and its llms.txt to be deployed before generating the "
            "site-root index"
        )

    base = f"{site_url}/{version}"
    lines = [
        _REL_LINK_RE.sub(lambda m: f"]({base}/{m.group(1)})", line)
        for line in src.read_text(encoding="utf-8").splitlines()
    ]

    version_links = _version_links(versions, site_url)
    if version_links:
        lines += ["", "## Other versions", *version_links]

    return "\n".join(lines).rstrip("\n") + "\n"


def _load_versions(gh_pages: Path) -> list:
    versions_file = gh_pages / "versions.json"
    if not versions_file.exists():
        raise FileNotFoundError(
            f"{versions_file} not found; expected mike to have published the "
            "version index before generating the site-root llms.txt"
        )
    return json.loads(versions_file.read_text(encoding="utf-8"))


def _resolve_alias(versions: list, alias: str) -> str:
    """Resolve a moving alias (e.g. latest-stable) to its concrete version, so
    the site-root index links at stable permalinks rather than the alias."""
    for entry in versions:
        if alias in (entry.get("aliases") or []):
            version = entry.get("version")
            if version:
                return version
    raise LookupError(f"no version in versions.json carries alias '{alias}'")


def _version_links(versions: list, site_url: str) -> list:
    links = []
    for entry in versions:
        version = entry.get("version")
        if not version:
            continue
        title = entry.get("title") or version
        aliases = entry.get("aliases") or []
        if aliases:
            title = f"{title} ({', '.join(aliases)})"
        links.append(f"- [{title}]({site_url}/{version}/llms.txt)")
    return links


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("gh_pages", help="path to the checked-out gh-pages worktree")
    parser.add_argument("--site-url", default=DEFAULT_SITE_URL)
    parser.add_argument("--default", dest="alias", default=DEFAULT_ALIAS)
    parser.add_argument(
        "--allow-missing-default",
        action="store_true",
        help="skip (exit 0) instead of failing when no version carries the "
        "default alias yet — e.g. when run from the dev pipeline before the "
        "first stable release",
    )
    args = parser.parse_args(argv)

    gh_pages = Path(args.gh_pages)
    try:
        content = build_root_llms(gh_pages, args.site_url, args.alias)
    except LookupError as exc:
        # The default alias is not published yet (e.g. dev deploy before the
        # first stable release). Skipping is fine; failing is not.
        if args.allow_missing_default:
            print(f"note: {exc}; skipping site-root llms.txt", file=sys.stderr)
            return 0
        print(f"error: {exc}", file=sys.stderr)
        return 1
    except FileNotFoundError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1

    out = gh_pages / "llms.txt"
    out.write_text(content, encoding="utf-8")
    print(f"wrote {out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
