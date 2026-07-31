# Vetix — one-shot SKILL security scanner
#
# Layout note: vetix/config.py resolves config as `Path(__file__).parent.parent / "config.yaml"`.
# With an editable install from /app, that resolves to /app/config.yaml, so we keep the
# source-tree layout (copy source to /app, WORKDIR /app, run via `uv run vetix`) and do
# NOT pip-install into site-packages (that would move the package and break config lookup).
FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim

ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    UV_NO_SYNC=1

WORKDIR /app

# Build-backend inputs first: hatchling reads `readme = "README.md"` and the dynamic
# version from vetix/__init__.py. uv.lock is gitignored (absent in a fresh clone), so
# it is never copied; `uv sync` (not --frozen) regenerates it.
COPY pyproject.toml README.md LICENSE ./
COPY vetix/ ./vetix/

# Template only — NOT the runtime config. The image deliberately ships no
# /app/config.yaml so credentials can never be baked in; a scan fails fast
# with FileNotFoundError if the user forgets to mount their config.
COPY example.config.yaml ./

RUN uv sync

ENTRYPOINT ["uv", "run", "vetix"]
# Safe default: prints usage and exits 0. Real scans always pass an explicit
# `scan -s /skills/<name>` (args after the image name replace this CMD).
CMD ["--help"]
