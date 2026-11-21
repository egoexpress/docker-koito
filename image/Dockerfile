# Vendored from github.com/gabehf/Koito at commit 9607b66 (pull request #382).
#
# Local changes against upstream:
#   - frontend stage pinned to node:22; node:latest now resolves to Node 26,
#     where Yarn 4 aborts with "onCancel handler was attached after the
#     promise settled"
#   - "--platform=$BUILDPLATFORM" dropped from the frontend FROM line so the
#     legacy Docker builder can build the image as well
#
# See README.md in this directory for provenance and update instructions.

FROM node:22 AS frontend

ARG KOITO_VERSION
ENV VITE_KOITO_VERSION=$KOITO_VERSION
ENV BUILD_TARGET=docker

WORKDIR /client
RUN npm install -g corepack
RUN corepack enable && corepack prepare yarn@4 --activate
COPY ./client .
RUN yarn install

RUN yarn run build

RUN find ./build/client -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.svg" \) -exec gzip -k -9 {} \;

FROM golang:1.25 AS backend

ARG KOITO_VERSION
ENV CGO_ENABLED=1
ENV GOOS=linux

WORKDIR /app

RUN apt-get update && \
	apt-get install -y libvips-dev pkg-config && \
	rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags "-X main.Version=$KOITO_VERSION" -o app ./cmd/api


FROM debian:bookworm-slim AS final

WORKDIR /app

RUN apt-get update && \
	apt-get install -y libvips42  && \
	rm -rf /var/lib/apt/lists/*

COPY --from=backend /app/app ./app
COPY --from=frontend /client/build ./client/build
COPY ./client/public ./client/public
COPY ./assets ./assets
COPY ./db ./db

EXPOSE 4110

ENTRYPOINT ["./app"]
