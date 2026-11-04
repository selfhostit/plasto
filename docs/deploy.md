# Deploy

A plasto app runs as one container on any Linux machine with Docker: a NAS, a
Raspberry Pi, a small server. Workers run next to it or on other machines.
Tailscale gives your devices an `https://` address for it without opening
anything to the internet.

```
 phone / laptop ──Tailscale──▶ tailscale sidecar ──▶ app (PocketBase + web app) ── pb_data/
                                                       ▲
                     worker on the same machine ───────┤  (records through the API)
                     worker on a Pi ──Tailscale────────┘
```

Two images are published for amd64, arm64 and armv7: `ghcr.io/selfhostit/plasto` (the
app) and `ghcr.io/selfhostit/plasto-worker` (the Python workers). A release tag like
`v1.2.0` publishes `:1.2.0`, `:1.2`, `:1` and `:latest` of both; installs follow `:latest`,
so they only update to a release. Every push to `main` publishes `:edge` and the commit
hash, for trying out what's next. Each image comes with a signed
[provenance attestation](#check-where-an-image-came-from).

## Install

On the machine, as a user that can run `docker`:

```sh
curl -fsSL https://raw.githubusercontent.com/selfhostit/plasto/main/deploy/install.sh | sh
```

This creates `~/plasto` with the compose files, a `.env` and a `pb_data` folder
for the database, asks for a Tailscale auth key (press Enter to skip, or set an
empty `TS_AUTHKEY=` to skip the question), and starts the app. Pass a folder to
install elsewhere: `... | sh -s /srv/plasto`.

Then open the app and create your account with the setup code the installer shows
(it is also in `docker compose logs app`); you become the owner. Only someone who can
see the machine has the code, so finish setup before or after making the app public,
whichever suits you.
Without Tailscale it is only reachable on the machine itself, at
http://localhost:8090 (the port is bound to `127.0.0.1` on purpose).

Settings live in `.env`: time zone, port, Tailscale, the worker account and the
image version. Change them and run `docker compose up -d` in the folder (or the
install script again).

## Remote access with Tailscale

1. In the [Tailscale admin console](https://login.tailscale.com/admin/dns), turn on
   **MagicDNS** and **HTTPS Certificates**.
2. Create an [auth key](https://login.tailscale.com/admin/settings/keys) and paste it
   when the install script asks. It is only used for the first sign-in; the
   sidecar keeps its login in `tailscale/`.
3. The app is now at `https://plasto.<your-tailnet>.ts.net` (the script prints it).
   The first visit can take a few seconds while the certificate is issued.

On a phone, install the Tailscale app, open that address and add it to the home
screen. Push notifications need this HTTPS address.

To add Tailscale later, set `TS_AUTHKEY` and uncomment the `COMPOSE_FILE` line in
`.env`, then run `docker compose up -d`. `TS_HOSTNAME` sets the name.

## Updates and versions

```sh
cd ~/plasto && docker compose pull && docker compose up -d
```

Running the install script again does the same and also refreshes the compose
files. Updates bring the newest release, listed with what changed in
[CHANGELOG.md](../CHANGELOG.md). To stay on a release line, or go back to a release, set
`PLASTO_IMAGE` (and `PLASTO_WORKER_IMAGE`) in `.env` to a tag, like
`ghcr.io/selfhostit/plasto:1` (every 1.x) or `ghcr.io/selfhostit/plasto:1.2.0`. Use `:edge`
to follow `main`. Database migrations run on start, so take a backup before going back
across one.

### Check where an image came from

Every published image has a signed attestation that says which workflow run, in which
repository and at which commit, built it. With the [GitHub CLI](https://cli.github.com):

```sh
gh attestation verify oci://ghcr.io/selfhostit/plasto:latest --repo selfhostit/plasto
```

It fails if the image was not built by this repository's CI, for example if someone pushed one by hand.

### Releasing

Add what changed under a new version in `CHANGELOG.md`, merge, then publish a GitHub
release with a tag like `v1.2.0` (Releases > Draft a new release) and paste the changelog
entry as its notes. CI tests it, publishes the images and attests them.

## Backups and restore

The app backs up the database and uploads every night at 03:00 and keeps the
last 7, in `pb_data/backups`. A backup on the same disk doesn't survive the disk,
so copy `pb_data` somewhere else too (Hyper Backup on a Synology, `rsync`, or S3
storage set in the admin UI under Settings > Backups). Encrypt any copy that leaves
the machine, and leave out the `tailscale` folder: it holds the machine's Tailscale
key, which lets whoever has it join your tailnet as that machine.

To restore, open the admin UI (`/_/`), go to Settings > Backups and restore one.
To move to another machine, stop the app, copy the whole folder, and start it
there.

## Workers

### Next to the app

1. In the admin UI, open the **workers** collection and add one with a name and a
   password.
2. In `.env`, set `PLASTO_WORKER` and `PLASTO_WORKER_PASSWORD` and uncomment
   `COMPOSE_PROFILES=worker`.
3. Run `docker compose up -d`.

It runs `workers/worker.py`, the starting point for your own worker. To run another
one, set `PLASTO_WORKER_SCRIPT` in `.env`, for example to `camera.py`. Every `.py`
file in `workers/` and every dependency you add with `uv add` is in the worker image.

### On another machine

Join the machine to your tailnet, then point the worker at the app's tailnet
address:

```sh
docker run -d --name worker --restart unless-stopped \
  -e PLASTO_URL=https://plasto.<your-tailnet>.ts.net \
  -e PLASTO_WORKER=camera -e PLASTO_WORKER_PASSWORD=... -e TZ=Europe/Athens \
  ghcr.io/selfhostit/plasto-worker python camera.py
```

## Synology NAS

Needs DSM 7.2 or later with **Container Manager** installed.

1. In Control Panel > Terminal & SNMP, enable SSH, then sign in with
   `ssh you@nas.local`.
2. Install into a shared folder, with `sudo` because Docker on DSM needs it:
   ```sh
   curl -fsSL https://raw.githubusercontent.com/selfhostit/plasto/main/deploy/install.sh | sudo sh -s /volume1/docker/plasto
   ```
   The data stays owned by your DSM user (`PUID` and `PGID` in `.env`), so you can
   see and back it up in File Station.
3. The containers show up in Container Manager, where you can see their logs and
   restart them. Update from SSH as above.

To serve the app on the internet instead of only your tailnet, add a rule under
Control Panel > Login Portal > Advanced > Reverse Proxy from your HTTPS host name
to `localhost:8090`. Never do this for apps that show cameras.

## Raspberry Pi

Use a Pi 4 or 5 with the 64-bit Raspberry Pi OS (Lite is enough).

```sh
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # then sign out and back in
```

Then run the install script as above to host the app on the Pi, or follow
[On another machine](#on-another-machine) to use the Pi only for workers, with
Tailscale installed on the Pi itself
(`curl -fsSL https://tailscale.com/install.sh | sh && sudo tailscale up`).

Hardware a worker needs, like a Hailo accelerator or a camera, is passed to its
container with `devices:` in compose (or `--device /dev/hailo0` with `docker run`).
Large services such as Frigate run as their own containers next to the worker.

## Check a deploy

`make smoke` builds both images, deploys them with `docker-compose.yml` in a
scratch folder, signs up an owner, runs one round of the worker and checks that
it reported in. CI runs the same script on every pull request.
