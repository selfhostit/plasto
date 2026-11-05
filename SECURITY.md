# Security policy

## Report a problem privately

Please don't open a public issue for a security problem. Use **Report a vulnerability** on
this repository's [Security tab](https://github.com/selfhostit/plasto/security/advisories/new),
<!-- TEMPLATE start -->
or write to **kalfasyan@pm.me**
<!-- TEMPLATE end -->
with which version (the image tag), what you found, how to reproduce it and what someone
could do with it.

## What happens next

- You get a reply within 7 days.
- A critical problem, such as reading or changing data that isn't yours or taking over the
  server, is fixed in a new release within 7 days of confirming it. Other problems are fixed
  in a release within 30 days.
- Once the fix is released, a [security advisory](https://github.com/selfhostit/plasto/security/advisories)
  says what was wrong, which versions were affected and how to update. It credits you,
  unless you'd rather not be named.
- There is no bug bounty.

## Supported versions

Fixes go into a new release of the newest version. Installs follow its major version
(`:1`), so `docker compose pull` or `selfhostit update` brings the fix; older releases
are not patched. See [docs/deploy.md](docs/deploy.md#updates-and-versions).

## How installs are protected

The "Security and maintenance" section of the [README](README.md#security-and-maintenance),
and for every selfhostit app, [Security and privacy](https://github.com/selfhostit/docs/blob/main/security.md).
