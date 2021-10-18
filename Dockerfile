# Build stage
FROM golang:1.16-alpine3.13 as gobuilder

WORKDIR /go/src/app

COPY . .

RUN CGO_ENABLED=0 go build -a -ldflags '-w -extldflags "-static"' -mod vendor .

# Final Stage
FROM busybox:1.33.1

WORKDIR /opt/sds

COPY --from=gobuilder /go/src/app/sigplot-data-service .

RUN mkdir logs

ENTRYPOINT [ "/opt/sds/sigplot-data-service" ]
