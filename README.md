# docker-koito

Docker Compose setup for [Koito](https://koito.io), a ListenBrainz-compatible
scrobbler, served behind [nginx-proxy](https://github.com/nginx-proxy/nginx-proxy)
with automatic Let's Encrypt certificates from
[acme-companion](https://github.com/nginx-proxy/acme-companion).

Koito keeps your listening history in a local SQLite database and exposes the
ListenBrainz API, so anything that scrobbles to ListenBrainz can scrobble to it.

## What this repository contains

| File                 | Purpose                                      |
| -------------------- | -------------------------------------------- |
| `docker-compose.yaml` | The Koito service and its persistent volume   |
| `.env.example`        | Template for the hostname and certificate email |

## Architecture

```
client ──HTTPS──▶ nginx-proxy ──HTTP:80──▶ koito:80 ──▶ /etc/koito/koito.db
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
$EDITOR .env        # set VIRTUAL_HOST and LETSENCRYPT_EMAIL
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
| `VIRTUAL_PORT`          | `80`                        | Required, see below                                          |
| `LETSENCRYPT_HOST`      | from `.env`                 | Domain to request a certificate for                          |
| `LETSENCRYPT_EMAIL`     | from `.env`                 | Let's Encrypt contact address                                |
| `KOITO_LISTEN_PORT`     | `80`                        | Port Koito binds inside the container                        |
| `KOITO_LOG_LEVEL`       | `warn`                      | `debug`, `info`, `warn`, `error`, `fatal`                    |
| `KOITO_CORS_ALLOWED_ORIGINS` | `*`                    | Allows browser clients from any origin to call the API        |

### Why `VIRTUAL_PORT` is required

The Koito image only `EXPOSE`s port `4110`. nginx-proxy resolves the upstream
port as `VIRTUAL_PORT`, otherwise the single exposed port, otherwise `80`. Since
`KOITO_LISTEN_PORT=80` makes Koito listen on 80 only, the proxy would target the
closed port 4110 and answer with `502 Bad Gateway`.

Alternatively, drop `KOITO_LISTEN_PORT` entirely: Koito then listens on its
default 4110, and nginx-proxy discovers that exposed port on its own.

Both `VIRTUAL_PORT` and `LETSENCRYPT_*` work unchanged with the current
acme-companion release. Newer deployments prefer `ACME_HOST` and `ACME_EMAIL`
instead of the two `LETSENCRYPT_*` variables.

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

## Updating

The image is pinned to a release tag. To move to a newer release, bump the tag
in `docker-compose.yaml` and recreate the container:

```sh
docker compose pull && docker compose up -d
```

Migrations run automatically on startup. Take a backup first.

## Troubleshooting

- **`502 Bad Gateway`** — the proxy cannot reach the container. Check
  `VIRTUAL_PORT=80` is present and that Koito is still listening on 80
  (`KOITO_LISTEN_PORT`).
- **`Could not find network named 'proxy_frontend'`** — the external network is
  missing; create it or start the proxy stack first.
- **No certificate issued** — DNS for `VIRTUAL_HOST` must resolve to the proxy
  host before acme-companion can validate the HTTP-01 challenge.
- **Login fails / `changeme` rejected** — the default user is only created on
  first startup. Koito can generate a new random password for the admin user:

  ```sh
  docker compose exec app ./app reset-password
  ```

  The new password is written to the container's standard output.

## Credits

Koito is developed by [gabehf](https://github.com/gabehf/Koito) and licensed
separately from this Compose setup. This repository only provides the
deployment configuration.
