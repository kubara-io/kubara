import re
import shutil
from pathlib import Path, PurePosixPath
from urllib.parse import urljoin

BLOCK = """
<blockquote class="sr-only" data-agent-docs-index="true" aria-hidden="true">
  <h2>Documentation Index</h2>
  <p>
    Fetch the complete documentation index at:
    <a href="/llms.txt">/llms.txt</a>
  </p>
  <p>
    Use this file to discover all available pages before exploring further.
  </p>
</blockquote>
""".strip()


LLMS_LINES = []
_BODY_RE = re.compile(r"(<body\b[^>]*>)", re.IGNORECASE)


def on_post_page(output, page, config):
    return _BODY_RE.sub(
        lambda m: f"{m.group(1)}\n{BLOCK}",
        output,
        count=1,
    )


def on_nav(nav, config, files):
    global LLMS_LINES

    LLMS_LINES = _walk_nav(
        nav.items,
        site_url=config.get("site_url", ""),
        use_directory_urls=config.get("use_directory_urls", True),
    )

    return nav


def on_post_build(config):
    docs_dir = Path(config["docs_dir"]).resolve()
    site_dir = Path(config["site_dir"]).resolve()
    use_directory_urls = config.get("use_directory_urls", True)

    for src in docs_dir.rglob("*.md"):
        rel = src.relative_to(docs_dir)

        if use_directory_urls:
            if rel.name == "index.md":
                dest = site_dir / rel.with_suffix(".md")
            else:
                dest = site_dir / rel.with_suffix("") / "index.md"
        else:
            dest = site_dir / rel

        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dest)

    (site_dir / "llms.txt").write_text(
        "\n".join(LLMS_LINES) + "\n",
        encoding="utf-8",
    )


def _published_md_path(src_uri: str, use_directory_urls: bool) -> str:
    p = PurePosixPath(src_uri)

    if not use_directory_urls:
        return p.as_posix()

    if p.name == "index.md":
        return p.as_posix()

    return (p.with_suffix("") / "index.md").as_posix()


def _to_url(site_url: str, path: str) -> str:
    if site_url:
        return urljoin(site_url.rstrip("/") + "/", path.lstrip("/"))

    return "/" + path.lstrip("/")


def _walk_nav(items, site_url: str, use_directory_urls: bool, depth: int = 0):
    lines = []

    for item in items:
        indent = "  " * depth
        children = getattr(item, "children", None)

        if children:
            title = getattr(item, "title", None)
            if title:
                lines.append(f"{indent}- {title}")

            lines.extend(
                _walk_nav(children, site_url, use_directory_urls, depth + 1)
            )
            continue

        file = getattr(item, "file", None)
        if not file or not getattr(file, "src_uri", None):
            continue

        md_path = _published_md_path(file.src_uri, use_directory_urls)
        md_url = _to_url(site_url, md_path)
        lines.append(f"{indent}- {item.title}: {md_url}")

    return lines
