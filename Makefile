GOCMD=CGO_ENABLED=0 go
GOFLAGS=-a -ldflags '-w -extldflags "-static"' -mod vendor

.PHONY: ui docker sds

all: ui sds

ui:
	npm --prefix ./ui/webapp install
	npm --prefix ./ui/webapp run build

sds:
	$(GOCMD) build $(GOFLAGS) cmd/sds/sigplot_data_service.go

docker:
	docker build -t sds:0.7 .
