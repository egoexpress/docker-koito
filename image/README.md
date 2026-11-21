# Vendored Koito source

This directory is the [Koito](https://github.com/gabehf/Koito) source tree at a
pinned commit, built into a custom image and published to GHCR.

| | |
| --- | --- |
| Upstream | https://github.com/gabehf/Koito |
| Commit | `9607b66` |
| Pull request | [#382 — feat: more complete listenbrainz api](https://github.com/gabehf/Koito/pull/382) |
| Base version | v0.3.2 |
| License | MIT, © Gabe Farrell — see [LICENSE](LICENSE) |

## Why

Upstream `v0.3.x` only implements two ListenBrainz endpoints, `submit-listens`
and `validate-token`. PR #382 adds the read side — `user/{user}/listens`,
`playing-now`, `listen-count`, `following`, the `stats/user/{user}/*` endpoints
and feedback — which scrobblers such as PanoScrobbler need to display history
and statistics from the server.

## Local changes to upstream

Only the `Dockerfile` differs:

- the frontend stage is pinned to `node:22`, because `node:latest` now resolves
  to Node 26, where Yarn 4 fails with `onCancel handler was attached after the
  promise settled`
- `--platform=$BUILDPLATFORM` is removed from the frontend `FROM` line, so the
  legacy Docker builder can build the image too (buildx handles it either way)

All Go and frontend sources are byte-identical to the upstream commit.

## Building locally

```sh
docker build -t koito:0.3.2-pr382 --build-arg KOITO_VERSION=0.3.2-pr382 image
```

## Updating

1. Fetch the new upstream ref, for example
   `git fetch https://github.com/gabehf/Koito pull/382/head`.
2. Replace every file in this directory except `Dockerfile` and this README with
   the tree at that commit.
3. Re-apply the two Dockerfile changes described above, update the commit in
   this README, and bump the image tag in `docker-compose.yaml` and
   `.github/workflows/publish-image.yml`.
4. Once upstream ships these endpoints in a release, delete this directory and
   go back to `gabehf/koito:<version>` in the Compose file.

## License

The vendored code keeps its upstream MIT license in [LICENSE](LICENSE). The MIT
license of this repository covers the Compose setup, not this directory.
