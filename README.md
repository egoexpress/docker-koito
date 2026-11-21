# docker-koito

[![validate](https://github.com/egoexpress/docker-koito/actions/workflows/validate.yml/badge.svg)](https://github.com/egoexpress/docker-koito/actions/workflows/validate.yml)

Docker Compose setup for [Koito](https://koito.io), a ListenBrainz-compatible
scrobbler, served behind [nginx-proxy](https://github.com/nginx-proxy/nginx-proxy)
with automatic Let's Encrypt certificates from
[acme-companion](https://github.com/nginx-proxy/acme-companion).

Koito keeps your listening history in a local SQLite database and exposes the
ListenBrainz API, so anything that scrobbles to ListenBrainz can scrobble to it.

## What this repository contains

| File                             | Purpose                                          |
| -------------------------------- | ------------------------------------------------ |
| `docker-compose.yaml`            | The Koito service and its persistent volume      |
| `.env.example`                   | Template for the hostname and certificate email  |
| `renovate.json`                  | Automated image updates for the Koito tag        |
| `LICENSE`                        | MIT license for this deployment configuration    |
| `.github/workflows/validate.yml` | CI check that the Compose file renders           |

## Architecture

```
client ──HTTPS──▶ nginx-proxy ──HTTP:4110──▶ koito:4110 ──▶ /etc/koito/koito.db
                       ▲
                       └── acme-companion (certificates)
```

Both proxy containers and this stack share the external Docker network
`proxy_frontend`. nginx-proxy picks up the `VIRTUAL_HOST` environment variable,
creates the virtual host for that domain, and acme-companion requests a
certificate for it.

## Requirements

- Docker with the Compose plugin
- A host with a public IPv4/IPv6 address, ports 80 and 443 reachable
- A DNS `A`/`AAAA` record for your domain pointing at that host
- A running nginx-proxy + acme-companion stack attached to a network named
  `proxy_frontend`

If you do not have such a proxy stack yet, this is the minimal shape it takes:

```yaml
services:
  nginx-proxy:
    image: nginxproxy/nginx-proxy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
      - certs:/etc/nginx/certs
      - vhost:/etc/nginx/vhost.d
      - html:/usr/share/nginx/html
    networks:
      - proxy_frontend

  acme-companion:
    image: nginxproxy/acme-companion
    restart: unless-stopped
    volumes_from:
      - nginx-proxy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - acme:/etc/acme.sh
    environment:
      - DEFAULT_EMAIL=admin@example.com
    networks:
      - proxy_frontend

volumes:
  certs:
  vhost:
  html:
  acme:

networks:
  proxy_frontend:
    name: proxy_frontend
```

The network has to exist before this stack starts, because it is declared
`external`. The proxy stack above creates it; to create it by hand:

```sh
docker network create proxy_frontend
```

## Setup

```sh
git clone git@github.com:egoexpress/docker-koito.git
cd docker-koito
cp .env.example .env
$EDITOR .env        # set VIRTUAL_HOST and ACME_EMAIL
docker compose up -d
```

Check the logs on first start — Koito runs its database migrations there:

```sh
docker compose logs -f
```

Once the certificate is issued, the instance is reachable at
`https://$VIRTUAL_HOST`.

### First login

On first startup Koito creates a user with the default credentials
`admin` / `changeme`. **Change the password immediately** after logging in.

The defaults can be overridden for the very first start with
`KOITO_DEFAULT_USERNAME` and `KOITO_DEFAULT_PASSWORD`; they have no effect once
the user exists.

## Configuration

| Variable                | Value                       | Notes                                                        |
| ----------------------- | --------------------------- | ------------------------------------------------------------ |
| `VIRTUAL_HOST`          | from `.env`                 | Domain nginx-proxy serves                                    |
| `ACME_HOST`             | from `.env`                 | Domain to request a certificate for                          |
| `ACME_EMAIL`            | from `.env`                 | Let's Encrypt contact address                                |
| `KOITO_LOG_LEVEL`       | `warn`                      | `debug`, `info`, `warn`, `error`, `fatal`                    |

CORS is left at Koito's default, which sends no CORS headers. That is all the
web client needs, since it is served from this same host. Only add
`KOITO_CORS_ALLOWED_ORIGINS` when a browser application on another domain calls
the API directly.

### How the proxy finds the web port

Koito listens on its default port `4110`, and the image declares exactly that
port with `EXPOSE`. Because it is the only exposed port, nginx-proxy uses it as
the upstream with no extra configuration.

If you ever set `KOITO_LISTEN_PORT` to another value, also set `VIRTUAL_PORT` to
match. nginx-proxy otherwise keeps targeting `4110` and answers with
`502 Bad Gateway`.

`ACME_HOST` and `ACME_EMAIL` are the current acme-companion variable names. Older
releases used `LETSENCRYPT_HOST` and `LETSENCRYPT_EMAIL`, which are deprecated but
still accepted.

The full list of Koito options is in the
[configuration reference](https://koito.io/reference/configuration/). Every
option can be appended to the `environment` list of the `app` service.

## Data and backups

Everything Koito writes lives in the named volume `docker-koito_data`, mounted
at `/etc/koito`:

- `koito.db` — the SQLite database with all listens
- `koito.db-wal`, `koito.db-shm` — SQLite write-ahead log files
- `import/` — import workspace

Because SQLite runs in WAL mode, stop the container before copying the database:

```sh
docker compose stop app
docker run --rm -v docker-koito_data:/data -v "$PWD":/backup \
  alpine tar czf /backup/koito-backup.tar.gz -C /data .
docker compose start app
```

## Logs

`docker compose logs -f` follows the container output. The service caps its
`json-file` logs at 10 MB across 3 files (30 MB total), so a long-running
instance cannot grow logs without bound on a host without global rotation.

## Healthcheck

The service probes Koito's readiness endpoint `/apis/web/v1/health`, which
answers `503` until the database store and HTTP server are up, and `200`
afterwards:

```sh
docker compose ps
docker inspect --format '{{.State.Health.Status}}' docker-koito-app-1
```

The probe uses `bash` and `/dev/tcp` because the image ships neither `curl` nor
`wget`. It honours `KOITO_LISTEN_PORT` and falls back to `4110`, so it keeps
working if you change the listen port.

`restart: unless-stopped` reacts to container exits, not to an `unhealthy`
state — Compose does not restart on a failed probe. The status is visible in
`docker compose ps`, in monitoring that reads container health, and in a proxy
stack that waits on `depends_on: condition: service_healthy`.

## Updating

The image is pinned to `gabehf/koito:v0.3`, the 0.3 release line. Patch
releases land in that tag and are picked up by `docker compose pull`; a new
minor or major release needs a tag bump.

[Renovate](https://docs.renovatebot.com/) opens that bump once the Renovate
GitHub app is installed on this repository. `renovate.json` keeps the tag at
major.minor precision (patch tags are ignored) and does not pin image digests.

To update by hand, bump the tag and recreate the container:

```sh
docker compose pull && docker compose up -d
```

Migrations run automatically on startup. Take a backup first.

## Validation

Every push to `main` and every pull request runs `docker compose config` in
GitHub Actions to confirm the file still renders. To run the same check locally:

```sh
VIRTUAL_HOST=koito.example.com ACME_EMAIL=admin@example.com docker compose config --quiet
```

## Troubleshooting

- **`502 Bad Gateway`** — the proxy cannot reach the container. The port Koito
  listens on (`KOITO_LISTEN_PORT`, default `4110`) must match the port
  nginx-proxy targets: the image's exposed port, or `VIRTUAL_PORT` when set.
- **`Could not find network named 'proxy_frontend'`** — the external network is
  missing; create it or start the proxy stack first.
- **Container shows `(unhealthy)`** — Koito is not answering its readiness
  endpoint. Check `docker compose logs app` for a failed migration or a startup
  error, and confirm that any `KOITO_LISTEN_PORT` override matches the port
  Koito actually binds.
- **No certificate issued** — DNS for `VIRTUAL_HOST` must resolve to the proxy
  host before acme-companion can validate the HTTP-01 challenge.
- **Login fails / `changeme` rejected** — the default user is only created on
  first startup. Koito can generate a new random password for the admin user:

  ```sh
  docker compose exec app ./app reset-password
  ```

  The new password is written to the container's standard output.

## License

The deployment configuration in this repository is released under the
[MIT License](LICENSE).

Koito itself is a separate project by [gabehf](https://github.com/gabehf/Koito)
with its own license; this repository only provides the Compose setup.
