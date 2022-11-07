# syntax=docker/dockerfile:1

## Build
FROM golang:1.19-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY internal/ ./internal

COPY .git .
RUN GIT_COMMIT=$(git describe --always --dirty) && \
  GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD) && \
  GIT_USER=$(git log -1 --pretty=format:'%ae') && \
  GIT_TAG=$(git tag --points-at HEAD) && \
  DATE=$(date +"%m-%d-%y") && \
  go build \
  -o /gotmail_exporter \
  -ldflags "-X github.com/prometheus/common/version.Version=${GIT_TAG}  -X github.com/prometheus/common/version.Revision=${GIT_COMMIT} -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} -X github.com/prometheus/common/version.BuildUser=${GIT_USER} -X github.com/prometheus/common/version.BuildDate=${DATE}"

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /gotmail_exporter /gotmail_exporter

EXPOSE 2112

USER nonroot:nonroot

ENTRYPOINT ["/gotmail_exporter"]
