FROM golang:1.13.8-alpine3.11

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...

RUN go install -v ./...

RUN mkdir logs

ENTRYPOINT [ "sigplot-data-service" ]
