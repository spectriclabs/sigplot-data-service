# syntax=docker/dockerfile:1

FROM node:22-alpine AS jsbuilder

WORKDIR /app

COPY ui/webapp/package.json ui/webapp/package-lock.json ./
RUN npm ci

COPY ui/webapp/ ./
RUN npm run build

FROM golang:1.26-alpine AS gobuilder

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor/ vendor/

COPY cmd/ cmd/
COPY internal/ internal/
COPY ui/*.go ui/
COPY --from=jsbuilder /app/dist ui/webapp/dist

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags '-s -w -extldflags "-static"' -tags ui -mod vendor -o /out/sigplot_data_service ./cmd/sds

FROM busybox:1.36

WORKDIR /opt/sds

RUN mkdir -p /opt/sds/sdscache /data && chown -R 65532:65532 /opt/sds /data

COPY --from=gobuilder /out/sigplot_data_service .
COPY --chown=65532:65532 sds_config.json ./sds_config.json

USER 65532:65532

EXPOSE 5055

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:5055/sds/fs || exit 1

ENTRYPOINT ["/opt/sds/sigplot_data_service"]
