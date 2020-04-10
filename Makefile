.PHONY: ui docker release

all:
	go build

ui:
	@cd ui && npm install
	@cd ui && npm run build
	@go-bindata-assetfs -prefix ui -modtime 1480000000 -tags ui ./ui/dist/...

release: ui
	go build -tags ui

docker:
	docker build -t sds:0.3 .
