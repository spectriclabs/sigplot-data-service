FROM golang:1.13.8-alpine3.11

RUN apk add --no-cache nodejs npm make git yarn

WORKDIR /go/src/app

RUN go get github.com/go-bindata/go-bindata/...

RUN go get github.com/elazarl/go-bindata-assetfs/...

COPY . .

RUN go get -d -v ./...

RUN rm -f bindata_assetfs.go && make release && go install -tags ui

RUN mkdir logs

ENTRYPOINT [ "sigplot-data-service" ]
