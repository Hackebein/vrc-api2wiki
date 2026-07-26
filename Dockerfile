# syntax=docker/dockerfile:1.7

FROM golang:1.26-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl unzip \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN bash scripts/fetch-depotdownloader.sh

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X github.com/Hackebein/vrc-api2wiki/pkg/mediawiki.buildVersion=${APP_VERSION} -X github.com/Hackebein/vrc-api2wiki/pkg/vrchat.buildVersion=${APP_VERSION}" \
  -o /out/vrc-api2wiki ./cmd/vrc-api2wiki \
  && mkdir -p /out/third_party/DepotDownloader \
  && cp third_party/DepotDownloader/DepotDownloader /out/third_party/DepotDownloader/ \
  && cp third_party/DepotDownloader/VERSION third_party/DepotDownloader/NOTICE /out/third_party/DepotDownloader/

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/vrc-api2wiki /app/vrc-api2wiki
COPY --from=build /out/third_party /app/third_party
# go.mod marker so FindRepoRoot works when WORKDIR is /app
RUN printf 'module github.com/Hackebein/vrc-api2wiki\n\ngo 1.26.1\n' > /app/go.mod
USER nobody
ENTRYPOINT ["/app/vrc-api2wiki"]
