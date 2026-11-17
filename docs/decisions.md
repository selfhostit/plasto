# plasto: stack decisions and build plan

plasto is a template repository for small self-hosted apps. A new app starts with
GitHub's **Use this template** button and a rename script. It ships with a
login, a phone-installable UI, a database, push notifications, scheduled jobs,
backups, Python workers and a one-command deploy. Then you add your own idea on top.

This document records what was decided and why.

plasto is only for **new** repositories. The four existing apps (splitwiz,
should-we, desto, vresto) were not built with it and will not be migrated. They
are reference material: they show which pieces keep getting rebuilt, and
splitwiz shows the stack working. The first apps to be built from plasto are a
camera surveillance app and a CV-based news/job crawler.

## What every app needs

The four reference apps each rebuilt the same pieces by hand:

| Need | splitwiz | should-we | desto | vresto |
|---|---|---|---|---|
| One Docker image, compose, `.env` | yes | yes | yes | yes |
| NAS deploy behind a reverse proxy | yes | yes | | |
| Login | per person | admin + join links | password | |
| Installable PWA | yes | yes | | |
| Notifications | Web Push | | Pushbullet | |
| Scheduled jobs | recurring + catch-up | | scheduling, chains | |
| Storage | SQLite (PocketBase) | JSON files | Redis + SQLite | files |
| CLI | | yes | yes | yes |

plasto provides all of these once, so an app only adds its own data model,
screens and workers.

## Decisions

### 1. Shape: a starter repo now, shared package later

Each app is its own repository, created from plasto. Once a piece (push,
backups, setup screen, worker kit) has been copied into two apps and stops
changing, it moves into a shared Go module or Python package that apps import.
We do not build a shared platform up front.

### 2. Core: PocketBase + Svelte

- **Backend:** PocketBase used as a Go framework (not the prebuilt binary), so
  each app compiles to a single static binary with its own hooks, routes and
  migrations. It gives us the SQLite database, auth, file storage, realtime
  subscriptions, an admin UI, and backups.
- **Frontend:** Svelte 5 + TypeScript + Tailwind 4, built with Vite into static
  files that PocketBase serves. It is a PWA with a manifest, service worker,
  install prompt, dark mode and Web Push.
- **Why:** splitwiz proved it. It is one container, about 30 MB of RAM at rest,
  multi-arch builds without CGO, and it covers most of what the other apps built by hand.
  NiceGUI remains a fine choice for quick internal tools but is not the template's base.

### 3. Heavy work: Python workers that talk to the PocketBase API

Anything that needs Python, such as ML, scraping, geodata or LLM calls, runs as a
**worker**. It is a separate process or container that logs in to PocketBase
with a dedicated service account and reads or writes records through the REST
API and realtime subscriptions.

- A worker can run next to the app or on another machine (the Pi).
- The app reacts to records, not to workers. A worker writes a record, a Go
  hook sees it, and the user gets a push notification. Workers never send
  notifications themselves.
- A job queue collection (workers claim and complete jobs) is added only when
  an app actually needs it.
- plasto ships a small Python worker kit (auth, retry, subscribe, upload a
  file, report health) and a worker skeleton, managed with `uv`.

### 4. Remote access: Tailscale

Apps are private to your own devices by default. The compose file includes an
optional Tailscale sidecar, so the app gets a `https://<app>.<tailnet>.ts.net`
address with a valid certificate. Web Push and PWA install need HTTPS, and this
provides it. Exposing an app publicly (DSM reverse proxy + DDNS, as splitwiz
does) stays documented as an option, but never for apps that touch cameras.

### 5. LLM provider: pluggable, decided per app

Workers that need an LLM (the crawler) go through one small interface with two
backends. One is the Claude API for quality, the other is a local model through
Ollama for privacy and zero cost. The choice is an environment variable.

### 6. New apps: "Use this template" + rename script

`scripts/rename.sh <name>` replaces `plasto` in the Go module path, image name,
env var prefix, PWA manifest and docs. `scripts/strip-demo.sh` removes the demo
feature.

## Known future apps, and how they map

### Surveillance (first real app after the template)

- **Pi 5 (16 GB) + Hailo:** runs Frigate (RTSP ingest, Hailo object detection,
  recordings, clips) and Mosquitto.
- **Worker on the Pi:** subscribes to Frigate's MQTT events. It writes an
  `event` record (camera, label, score, time, snapshot) into the app and filters
  by zone, label and quiet hours.
- **App on the NAS:** stores events, sends push with the snapshot, shows a
  timeline, links to the clip in Frigate, and lets you mute a camera or label.
- Everything stays on the tailnet.

### Crawler (news, jobs, ... scored against your CV)

- A scheduled worker fetches sources (RSS, job boards, sites) and extracts text.
- The LLM interface scores each item against the CV and your interests.
- The app stores items, lets you thumb items up or down to tune scoring, and sends a
  daily digest push.

## What the template contains

```
plasto/
  backend/            Go app on PocketBase
    main.go
    app/              hooks, routes, push, backups, scheduler, setup
    migrations/       collections as code
  web/                Svelte 5 + Tailwind PWA
  workers/
    plasto_worker/    Python worker kit (uv package)
    example/          demo worker
  deploy/
    docker-compose.yml
    docker-compose.tailscale.yml
    install.sh
  docs/               NAS, Pi, Tailscale, "build your first feature"
  scripts/            rename.sh, strip-demo.sh
  Makefile            make dev / test / build
  .github/workflows/  CI, multi-arch image to GHCR
```

**Built in from day one**

- First-run setup screen (create the owner account), with users and roles, and a
  People screen where the owner adds people and gives them a new password
- Web Push with VAPID keys generated and stored on first start, plus a per-device
  opt-in page that explains why notifications can't be turned on
- A scheduler with catch-up for jobs missed while the server was off
- Nightly PocketBase backups with retention
- A health endpoint, `mem_limit`, TZ, and a `127.0.0.1` port by default
- A service-account flow for workers (create, rotate, revoke)
- CI covering Go tests, `svelte-check`, vitest, ruff, pytest, and an end-to-end smoke
  test that starts the image, runs the demo worker and checks the push record
- pre-commit with gitleaks, ruff and gofmt

**Demo feature (removable):** an `items` list. The demo worker adds an item
every few minutes, and a hook sends a push when one arrives. It exercises
every layer, and `scripts/strip-demo.sh` deletes it.

## Build plan

1. Skeleton: a Go module, PocketBase serving a Vite build, `make dev` with hot
   reload for both, and CI.
2. Auth and setup: the first-run owner screen, login, the PWA shell and dark mode.
3. Push: VAPID, subscriptions, the settings page and a notify helper for hooks.
4. Scheduler and backups.
5. Worker kit and the demo worker, with the service-account flow.
6. Deploy: a multi-arch image on GHCR, compose, the Tailscale sidecar, `install.sh`,
   and NAS and Pi docs.
7. Template ergonomics: `rename.sh`, `strip-demo.sh`, and a "build your
   first feature" guide.
8. Mark the repository as a GitHub template.

Each step is one PR that leaves the template working.
