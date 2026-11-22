# The worker image: the plasto_worker kit, its locked dependencies and the
# workers in this folder. It runs worker.py unless you give it another command:
#   docker run ... ghcr.io/selfhostit/plasto-worker python my_worker.py

# --- build: resolve the locked dependencies and build the kit's wheel ---
# Base images and uv are pinned by digest; Dependabot moves the digests weekly.
# uv only runs on the build machine; its image has no armv7 build, so it is taken
# for the build platform instead of each target one.
FROM --platform=$BUILDPLATFORM ghcr.io/astral-sh/uv:0.13.9@sha256:d3b601f0c7b118bc1753febf7c41cc80981df678445e0254284eecf84db79a7d AS uv
FROM --platform=$BUILDPLATFORM python:3.15-slim@sha256:65a65fc7b30b06cb6df4930fd4650697cb23f4c2698ab81fc87e4b031bc36928 AS build
COPY --from=uv /uv /usr/local/bin/uv
WORKDIR /src
COPY pyproject.toml uv.lock README.md ./
COPY src/ src/
RUN uv export --locked --no-dev --no-emit-project -o requirements.txt \
    && uv build --wheel -o dist

# --- runtime ---
FROM python:3.15-slim@sha256:65a65fc7b30b06cb6df4930fd4650697cb23f4c2698ab81fc87e4b031bc36928 AS runtime
ENV PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1
# Take Debian's security fixes now rather than waiting for the base image to be rebuilt.
RUN apt-get update \
    && apt-get upgrade -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/requirements.txt /src/dist/ /tmp/build/
# requirements.txt carries a hash for every package, so pip refuses anything else.
RUN pip install --no-cache-dir --require-hashes -r /tmp/build/requirements.txt \
    && pip install --no-cache-dir --no-deps /tmp/build/*.whl \
    && rm -rf /tmp/build \
    && useradd -u 1000 -m app
WORKDIR /app
COPY --chown=app:app *.py ./
USER app
CMD ["python", "worker.py"]
