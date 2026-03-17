# Introduction
https://www.mkdocs.org/getting-started/

# Getting Started
We use [uv](https://docs.astral.sh/uv/) for incredibly fast Python dependency and environment management. It requires Python 3.14.3, but **`uv` will automatically fetch and install this version for you** if it's missing from your system.

## Change directory to mkdocs directory
```bash
cd docs
```

## Install uv (if not already installed)
For macOS/Linux, you can use the standalone installer:
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```
*(Alternatively, check the [official installation guide](https://docs.astral.sh/uv/getting-started/installation/) for Windows or Homebrew instructions).*

## Run mkdocs
You don't even need to activate the virtual environment shell manually. You can run `mkdocs` directly through `uv`:
```bash
make serve
```
*(Note: If you prefer to activate the shell manually to run raw commands, use `source .venv/bin/activate` on Mac/Linux or `.venv\Scripts\activate` on Windows).*

## See live rendering
http://127.0.0.1:8000/


## Additional Styling Options:
https://squidfunk.github.io/mkdocs-material/reference/


### Used dependencies
These are now managed via `pyproject.toml` and locked in `uv.lock`:
- mike
- mkdocs
- mkdocs-material
- mkdocs-plugin-open-external-links-in-new-tab
