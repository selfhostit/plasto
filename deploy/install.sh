#!/bin/sh
# Installs or updates plasto on this machine with Docker Compose.
#
#   curl -fsSL https://raw.githubusercontent.com/selfhostit/plasto/main/deploy/install.sh | sh
#
# or, from a clone, `sh deploy/install.sh [folder]`. It sets up a folder
# (~/plasto by default) with the compose files and a .env, asks for a Tailscale
# auth key the first time (or takes TS_AUTHKEY from the environment), and starts
# the app. Run it again to update; it keeps your
# .env and data and replaces the compose files, so keep your settings in .env.
set -eu

app=plasto
repo=${PLASTO_REPO:-selfhostit/plasto}
source=${PLASTO_SOURCE:-https://raw.githubusercontent.com/$repo/main/deploy}
dir=${1:-${PLASTO_DIR:-$HOME/$app}}
files="docker-compose.yml docker-compose.tailscale.yml .env.example"

say() { printf '\033[1m%s\033[0m\n' "$*"; }
die() { printf 'Error: %s\n' "$*" >&2; exit 1; }

# interactive: true when there is a terminal to ask questions on (even with curl | sh)
interactive() { (: </dev/tty) 2>/dev/null; }

# setenv KEY VALUE: sets KEY in .env, uncommenting it if needed
setenv() {
	awk -v k="$1" -v v="$2" '
		$0 ~ "^#?" k "=" { print k "=" v; done = 1; next }
		{ print }
		END { if (!done) print k "=" v }
	' .env >.env.tmp && mv .env.tmp .env
}

fetch() {
	if command -v curl >/dev/null; then curl -fsSL "$1"; else wget -qO- "$1"; fi
}

command -v docker >/dev/null || die "Docker is not installed: https://docs.docker.com/engine/install/"
docker compose version >/dev/null 2>&1 || die "Docker Compose v2 is missing: https://docs.docker.com/compose/install/linux/"
docker info >/dev/null 2>&1 || die "cannot talk to Docker; run this as a user in the docker group, or with sudo"

say "Installing $app in $dir"
mkdir -p "$dir"
here=$(dirname "$0")
for f in $files; do
	if [ -f "$here/install.sh" ] && [ "$(cd "$here" && pwd)" != "$(cd "$dir" && pwd)" ]; then
		cp "$here/$f" "$dir/$f"
	elif [ ! -f "$here/install.sh" ]; then
		fetch "$source/$f" >"$dir/$f.tmp" || die "could not download $source/$f"
		mv "$dir/$f.tmp" "$dir/$f"
	fi
done
cd "$dir"

if [ ! -f .env ]; then
	cp .env.example .env
	tz=$(cat /etc/timezone 2>/dev/null || readlink /etc/localtime 2>/dev/null | sed 's|.*zoneinfo/||' || true)
	[ -n "$tz" ] && setenv TZ "$tz"
	# the data belongs to you, also when you run this with sudo
	uid=${SUDO_UID:-$(id -u)}
	if [ "$uid" != 0 ]; then
		setenv PUID "$uid"
		setenv PGID "${SUDO_GID:-$(id -g)}"
	fi
	setenv TS_HOSTNAME "$(basename "$dir")"

	# Asks only when TS_AUTHKEY is not set; set but empty means no Tailscale.
	key=${TS_AUTHKEY:-}
	if [ -z "${TS_AUTHKEY+set}" ] && interactive; then
		echo
		echo "Tailscale gives the app an https:// address that only your devices can reach,"
		echo "which phones need to install it and get notifications. Paste an auth key from"
		echo "https://login.tailscale.com/admin/settings/keys, or press Enter to skip."
		printf 'Tailscale auth key: '
		read -r key </dev/tty || key=
	fi
	if [ -n "$key" ]; then
		setenv TS_AUTHKEY "$key"
		setenv COMPOSE_FILE docker-compose.yml:docker-compose.tailscale.yml
	fi
fi

# The containers run as PUID:PGID, so the data folders must belong to them.
mkdir -p pb_data tailscale
if [ "$(id -u)" = 0 ]; then
	uid=$(sed -n 's/^PUID=//p' .env)
	gid=$(sed -n 's/^PGID=//p' .env)
	chown "${uid:-1000}:${gid:-1000}" pb_data
fi

say "Starting $app"
docker compose pull --quiet
docker compose up -d --remove-orphans

port=$(sed -n 's/^PORT=//p' .env)
port=${port:-8090}
for _ in $(seq 1 30); do
	fetch "http://127.0.0.1:$port/api/health" >/dev/null 2>&1 && break
	sleep 1
done
fetch "http://127.0.0.1:$port/api/health" >/dev/null 2>&1 ||
	die "the app did not start; see: cd $dir && docker compose logs app"

echo
say "$app is running."
echo "  On this machine:  http://localhost:$port"
if grep -q '^COMPOSE_FILE=.*tailscale' .env; then
	name=""
	for _ in $(seq 1 30); do
		name=$(docker compose exec -T tailscale tailscale status --json --peers=false 2>/dev/null |
			sed -n 's/.*"DNSName": *"\([^"]*\)\.".*/\1/p' | head -n 1)
		[ -n "$name" ] && break
		sleep 1
	done
	if [ -n "$name" ]; then
		echo "  On your tailnet:  https://$name"
	else
		echo "  Tailscale is not signed in yet; see: cd $dir && docker compose logs tailscale"
	fi
fi
echo "  Settings:         $dir/.env (then run this script again)"
echo "  Open the app and create your account; the first person becomes the owner."
