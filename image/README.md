# Vendored Koito source

This directory is the [Koito](https://github.com/gabehf/Koito) source tree at a
pinned commit, built into a custom image and published to GHCR.

| | |
| --- | --- |
| Upstream | https://github.com/gabehf/Koito |
| Commit | `36ca617` |
| Pull request | [#382 — feat: more complete listenbrainz api](https://github.com/gabehf/Koito/pull/382) |
| Base version | v0.3.2 |
| License | MIT, © Gabe Farrell — see [LICENSE](LICENSE) |

## Why

Upstream `v0.3.x` only implements two ListenBrainz endpoints, `submit-listens`
and `validate-token`. PR #382 adds the read side — `user/{user}/listens`,
`playing-now`, `listen-count`, `following`, the `stats/user/{user}/*` endpoints
and feedback — which scrobblers such as PanoScrobbler need to display history
and statistics from the server. The last commit on the PR, `36ca617`, also
returns `started_at` for the currently playing track.

## Local changes to upstream

`engine/handlers/lbz_read.go` comes from PR #382 and reads the now-playing entry
from `memkv`. Commit `36ca617` changed that entry from a bare track ID to
`catalog.NowPlayingState`, so `nowPlayingState` there asserts the new type and
uses `state.TrackID`.

The `Dockerfile` differs as well:

- the frontend stage is pinned to `node:22`, because `node:latest` now resolves
  to Node 26, where Yarn 4 fails with `onCancel handler was attached after the
  promise settled`
- `--platform=$BUILDPLATFORM` is removed from the frontend `FROM` line, so the
  legacy Docker builder can build the image too (buildx handles it either way)

Apart from that, the Go and frontend sources are byte-identical to the upstream
commit.

## Building locally

```sh
docker build -t koito:0.3.2-pr382.2 --build-arg KOITO_VERSION=0.3.2-pr382.2 image
```

## Updating

1. Fetch the new upstream ref, for example
   `git fetch https://github.com/gabehf/Koito pull/382/head`.
2. Replace every file in this directory except `Dockerfile` and this README with
   the tree at that commit.
3. Re-apply the local changes described above, update the commit in this README,
   and bump the image tag in `docker-compose.yaml` and
   `.github/workflows/publish-image.yml`.
4. Once upstream ships these endpoints in a release, delete this directory and
   go back to `gabehf/koito:<version>` in the Compose file.

## License

The vendored code keeps its upstream MIT license in [LICENSE](LICENSE). The MIT
license of this repository covers the Compose setup, not this directory.
