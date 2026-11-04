<div align="center">

<img src="web/public/icons/icon.svg" width="88" alt="">

# plasto

**A template for self-hosted apps: from an idea to an app on your phone,<br>
running in one small container on your own hardware.**

[![CI](https://github.com/selfhostit/plasto/actions/workflows/ci.yml/badge.svg)](https://github.com/selfhostit/plasto/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/github/license/selfhostit/plasto)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/selfhostit/plasto?filename=backend%2Fgo.mod&logo=go&logoColor=white)](backend/go.mod)
[![PocketBase](https://img.shields.io/badge/PocketBase-0.40-b8dbe4?logo=pocketbase&logoColor=black)](https://pocketbase.io)
[![Svelte 5](https://img.shields.io/badge/Svelte-5-ff3e00?logo=svelte&logoColor=white)](https://svelte.dev)
[![Tailwind CSS 4](https://img.shields.io/badge/Tailwind_CSS-4-38bdf8?logo=tailwindcss&logoColor=white)](https://tailwindcss.com)
[![Python workers](https://img.shields.io/badge/workers-Python_3.12+-3776ab?logo=python&logoColor=white)](workers/README.md)
[![Docker images](https://img.shields.io/badge/ghcr.io-amd64_%7C_arm64_%7C_armv7-2496ed?logo=docker&logoColor=white)](https://github.com/orgs/selfhostit/packages?repo_name=plasto)

[![Use this template](https://img.shields.io/badge/Use_this_template-2ea44f?style=for-the-badge&logo=github)](https://github.com/selfhostit/plasto/generate)
[![Buy me a coffee](https://img.shields.io/badge/Buy_me_a_coffee-ffdd00?style=for-the-badge&logo=buymeacoffee&logoColor=black)](https://buymeacoffee.com/kalfasyan)

<img src="docs/images/setup.png" width="230" alt="The first-run screen, where the first person creates the owner account">&nbsp;
<img src="docs/images/home-light.png" width="230" alt="The home screen with the live demo list, in light mode">&nbsp;
<img src="docs/images/home-dark.png" width="230" alt="The same home screen in dark mode">

</div>

plasto gives every new app the parts that keep getting rebuilt: sign-in and roles,
a database with an admin UI, push notifications, scheduled jobs, backups, Python
workers and a one-command deploy. You add your own idea on top.

It is built on [PocketBase](https://pocketbase.io) (Go, SQLite, auth, realtime,
admin UI) and [Svelte 5](https://svelte.dev) with
[Tailwind](https://tailwindcss.com). See [docs/decisions.md](docs/decisions.md)
for why.

<!-- TEMPLATE start (rename.sh removes this section from new apps) -->
## Start a new app

1. On GitHub, click **Use this template** and create your app's repository.
2. Clone it and run `scripts/rename.sh`: plasto becomes your repository's name in
   the code, images, env vars and docs.
3. Run `scripts/strip-demo.sh` to remove the demo, or keep it as a reference for now.
4. Follow [Build your first feature](docs/first-feature.md).

<!-- TEMPLATE end -->

## Quick start

You need [Go](https://go.dev/dl/) 1.27+, [Node](https://nodejs.org) 22+ and
[uv](https://docs.astral.sh/uv/) (for the Python workers).

```sh
make setup   # install dependencies
make dev     # backend on :8090, web app with live reload on http://localhost:5173
```

Open the app and create your account with the setup code that `make dev` prints:
you become the **owner** and can also sign in to the admin UI at http://localhost:8090/_/ with the same
email and password. Add more people there, in the `users` collection.

## What you get

- **First-run setup and sign-in.** The first account is made on a setup screen, with a
  one-time code from the installer or the app's log, so nobody else can claim a fresh install.
- **Users and roles.** The owner manages everyone; members can see each other and
  edit themselves. Nobody can sign themselves up.
- **Push notifications** straight from your server, with no account or third-party
  service. Each person turns them on per device; send one from Go with
  `push.ToUser(app, userID, push.Message{Title: "…", URL: "/…"})`. Phones need the
  app on HTTPS.
- **Scheduled jobs that don't get skipped.** `jobs.MustAdd(app, "digest", "0 7 * * *", fn)`
  runs on a cron schedule in your time zone (`TZ`). If the server was off when a job
  was due, it runs once when the server starts again. Every run and its error are
  listed in the admin UI under the `jobs` collection.
- **Nightly backups** of the database and uploads at 03:00, keeping the last 7, in
  `pb_data/backups`. Change the schedule or add S3 storage in the admin UI under
  Settings > Backups.
- **Python workers** for the heavy lifting (ML, scraping, LLMs), on this machine or
  another. They sign in with a worker account and read and write records through
  the API; see [workers/README.md](workers/README.md).
<!-- DEMO start -->
- **A demo** that uses every piece: a live items list on the home screen, a worker
  that adds to it, and a push notification for each new item.
  `scripts/strip-demo.sh` removes it.
<!-- DEMO end -->
- **An installable app.** It installs to a phone's home screen or a desktop, opens
  offline from its cached shell, and follows the system's light or dark mode.

## How it fits together

```mermaid
flowchart LR
  phone["Phone or laptop<br/>installable web app"]
  subgraph box["Your NAS, Pi or server"]
    app["Go app on PocketBase<br/>API, auth, realtime, admin UI"]
    db[("SQLite and files<br/>pb_data/")]
    jobs["Scheduled jobs<br/>and nightly backups"]
    app --- db
    app --- jobs
  end
  worker["Python workers<br/>ML, scraping, LLMs"]
  phone -- "HTTPS over Tailscale" --> app
  worker -- "records through the API" --> app
  app -. "Web Push" .-> phone
```

Everything a person sees is the web app, served by the Go app. Heavy work runs in
Python workers, here or on another machine, which write records; Go hooks react to
those records and notify people.

## Deploy

On any Linux machine with Docker (a NAS, a Raspberry Pi, a server):

```sh
curl -fsSL https://raw.githubusercontent.com/selfhostit/plasto/main/deploy/install.sh | sh
```

It asks for a Tailscale auth key, so your devices reach the app at
`https://plasto.<your-tailnet>.ts.net`, and starts it with Docker Compose. See
[docs/deploy.md](docs/deploy.md) for updates, backups, workers, Synology and the Pi.

## Layout

```
backend/            Go app on PocketBase
  main.go           entry point
  app/              routes, hooks and jobs (start here)
  jobs/             scheduled jobs with catch-up
  push/             Web Push: keys, subscriptions, sending
  migrations/       database schema as code
web/                Svelte 5 + Tailwind web app
  src/App.svelte    picks setup, sign-in or the app
  src/components/   screens; Home.svelte is where your app starts
  src/lib/          shared code (API client, session, install) and its tests
  public/           manifest, service worker and icons
workers/            Python worker kit, your workers, their tests and image
deploy/             compose files, Tailscale sidecar, install script, smoke test
docs/               decisions, the deploy guide and the first-feature guide
Dockerfile          the app's multi-arch image
Makefile            run `make` to see every command
```

## Commands

| Command      | What it does                                          |
| ------------ | ----------------------------------------------------- |
| `make dev`   | Run everything with live reload                       |
| `make check` | Format, lint and type-check                           |
| `make test`  | Run the Go, web and worker tests                      |
| `make build` | Build `dist/plasto` with the web app next to it       |
| `make run`   | Build the Docker image and run it on :8090            |
| `make smoke` | Deploy both images with Compose and test end to end   |

Pushes to `main` publish `ghcr.io/selfhostit/plasto` and
`ghcr.io/selfhostit/plasto-worker` for amd64, arm64 and armv7; a `v*` tag
publishes that version.

## Security and maintenance

- **Not on the internet:** the container listens on `127.0.0.1` only; reach it through
  Tailscale or your own reverse proxy with HTTPS (`deploy/docker-compose.yml`).
- **Locked-down container:** runs as a normal user, drops every Linux capability and
  can't gain new ones.
- **Nobody claims a new install before you:** the first account needs the one-time setup
  code the install prints (`backend/app/setup.go`).
- **Sign-ins are rate limited** (10 a minute, `backend/migrations/*_rate_limits.go`), and the
  web app sends a Content Security Policy (`backend/app/headers.go`).
<!-- TEMPLATE start -->
- **Apps made from plasto start with all of the above,** and the template's own CI builds an
  app with `scripts/rename.sh` and `scripts/strip-demo.sh` and runs its tests.
<!-- TEMPLATE end -->
- **Tested on every change:** Go, web and Python worker tests and an end-to-end run of the published images
  (`make smoke`) in [CI](.github/workflows/ci.yml); Actions are pinned to commit hashes and
  Dependabot keeps Go, npm, Python, Docker and Actions dependencies current.
- **No analytics or tracking,** and no account anywhere else: the data stays in `pb_data` on your
  machine, with a nightly backup kept there.

<!-- TEMPLATE start -->
Found a security problem? Please write to kalfasyan@pm.me rather than opening an issue
([security policy](https://github.com/selfhostit/.github/blob/main/SECURITY.md)). [Security and privacy](https://github.com/selfhostit/docs/blob/main/security.md) covers every selfhostit app:
what protects an install, what you are trusting and what is not done yet.
<!-- TEMPLATE end -->

## Support

If plasto saves you time, you can [buy me a coffee](https://buymeacoffee.com/kalfasyan).

## License

MIT
