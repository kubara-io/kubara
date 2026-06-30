"""MkDocs hook: publish agent-friendly documentation artifacts.

For every build it does three things:

1. Advertises a machine-readable documentation index via a
   ``<link rel="alternate">`` in each page's ``<head>``. The href is relative,
   so it keeps resolving under mike's versioned (``/<version>/``) deploys.
2. Copies the raw Markdown sources next to the rendered HTML, so the links in
   ``llms.txt`` resolve to agent-friendly Markdown instead of HTML.
3. Generates a per-version ``llms.txt`` (llmstxt.org shape) whose links are
   relative to the version root, which makes it correct for whichever version
   it is served under.

The site-root ``/llms.txt`` (the conventional discovery path, like
``robots.txt``) is a separate concern: mike leaves the site root as a redirect
stub, so it is generated in CI after ``mike set-default`` — see
``docs/scripts/build_root_llms.py``.
"""

import shutil
from pathlib import Path, PurePosixPath
from urllib.parse import quote

from mkdocs.utils import get_relative_url

LLMS_TXT = "llms.txt"

# Key under which the rendered llms.txt is stashed on the config between the
# on_nav (where the nav tree is available) and on_post_build (where the site
# directory is written) events. Using config avoids module-global state that
# could leak across builds in long-lived `mkdocs serve` sessions.
CONFIG_KEY = "_agent_docs_llms_txt"


def on_nav(nav, config, files):
    config[CONFIG_KEY] = _build_llms_txt(nav, config)
    return nav


def on_post_page(output, page, config):
    href = get_relative_url(LLMS_TXT, page.url)
    link = (
        '<link rel="alternate" type="text/plain" '
        f'title="LLM-friendly documentation index" href="{href}">'
    )
    return output.replace("</head>", f"{link}\n</head>", 1)


def on_post_build(config):
    docs_dir = Path(config["docs_dir"]).resolve()
    site_dir = Path(config["site_dir"]).resolve()
    use_directory_urls = config.get("use_directory_urls", True)

    for src in docs_dir.rglob("*.md"):
        rel = src.relative_to(docs_dir)

        if use_directory_urls and rel.name != "index.md":
            dest = site_dir / rel.with_suffix("") / "index.md"
        else:
            dest = site_dir / rel

        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dest)

    (site_dir / LLMS_TXT).write_text(
        config.get(CONFIG_KEY, ""),
        encoding="utf-8",
    )


def _build_llms_txt(nav, config) -> str:
    use_directory_urls = config.get("use_directory_urls", True)

    lines = [f"# {config['site_name']}", ""]

    summary = (config.get("site_description") or "").strip()
    if summary:
        lines += [f"> {summary}", ""]

    # Top-level leaf pages (e.g. Home) have no section to live under; emit them
    # as a short lead list before the first `##` heading.
    preamble = []
    sections = []  # list of (title, [bullet, ...])

    for item in nav.items:
        if getattr(item, "children", None):
            bullets = _section_links(item.children, use_directory_urls)
            if bullets:
                sections.append((item.title, bullets))
            continue

        bullet = _nav_link(item, use_directory_urls)
        if bullet:
            preamble.append(bullet)

    if preamble:
        lines += [*preamble, ""]

    for title, bullets in sections:
        lines += [f"## {title}", *bullets, ""]

    return "\n".join(lines).rstrip("\n") + "\n"


def _section_links(items, use_directory_urls, prefix="") -> list:
    """Flatten a nav subtree into bullets, keeping sub-section context as a
    breadcrumb prefix (e.g. ``SSO > How to add SSO``). The separator is kept
    ASCII so the index renders correctly regardless of how a consumer guesses
    the charset of the served text/plain file."""
    bullets = []

    for item in items:
        children = getattr(item, "children", None)
        if children:
            bullets += _section_links(
                children, use_directory_urls, f"{prefix}{item.title} > "
            )
            continue

        bullet = _nav_link(item, use_directory_urls, prefix)
        if bullet:
            bullets.append(bullet)

    return bullets


def _nav_link(item, use_directory_urls, prefix="") -> str | None:
    title = getattr(item, "title", None)
    if not title:
        return None
    # Escape brackets so a nav title containing `[`/`]` can't break the
    # generated Markdown link.
    title = f"{prefix}{title}".replace("[", r"\[").replace("]", r"\]")

    # Internal pages: link to the published raw Markdown, relative to the
    # version root (== the directory llms.txt is served from). External nav
    # entries (mkdocs Link items) carry an absolute URL and no file.
    file = getattr(item, "file", None)
    if file and getattr(file, "src_uri", None):
        url = quote(_published_md_path(file.src_uri, use_directory_urls), safe="/")
    else:
        url = getattr(item, "url", None)

    if not url:
        return None
    return f"- [{title}]({url})"


def _published_md_path(src_uri: str, use_directory_urls: bool) -> str:
    p = PurePosixPath(src_uri)

    if not use_directory_urls or p.name == "index.md":
        return p.as_posix()

    return (p.with_suffix("") / "index.md").as_posix()
