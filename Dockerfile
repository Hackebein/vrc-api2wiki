# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/Hackebein/vrc-api2wiki/pkg/mediawiki.buildVersion=${APP_VERSION} -X github.com/Hackebein/vrc-api2wiki/pkg/vrchat.buildVersion=${APP_VERSION}" -o /out/vrc-api2wiki ./cmd/vrc-api2wiki

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=build /out/vrc-api2wiki /app/vrc-api2wiki
USER nonroot:nonroot
ENTRYPOINT ["/app/vrc-api2wiki"]
