SDS=cmd/sigplot_data_service.go
FLAGS=-mod vendor

sds:
	go build ${FLAGS} ${SDS}

run:
	go run ${FLAGS} ${SDS}

test:
	go test

clean:
	rm -f sigplot_data_service
