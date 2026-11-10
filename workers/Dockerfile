# The worker image: the plasto_worker kit, its locked dependencies and the
# workers in this folder. It runs worker.py unless you give it another command:
#   docker run ... ghcr.io/selfhostit/plasto-worker python my_worker.py

# --- build: resolve the locked dependencies and build the kit's wheel ---
FROM --platform=$BUILDPLATFORM python:3.15-slim AS build
RUN pip install --no-cache-dir uv==0.11.32
WORKDIR /src
COPY pyproject.toml uv.lock README.md ./
COPY src/ src/
RUN uv export --locked --no-dev --no-emit-project -o requirements.txt \
    && uv build --wheel -o dist

# --- runtime ---
FROM python:3.15-slim AS runtime
ENV PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1
# Take Debian's security fixes now rather than waiting for the base image to be rebuilt.
RUN apt-get update \
    && apt-get upgrade -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/requirements.txt /src/dist/ /tmp/build/
RUN pip install --no-cache-dir -r /tmp/build/requirements.txt \
    && pip install --no-cache-dir --no-deps /tmp/build/*.whl \
    && rm -rf /tmp/build \
    && useradd -u 1000 -m app
WORKDIR /app
COPY --chown=app:app *.py ./
USER app
CMD ["python", "worker.py"]
