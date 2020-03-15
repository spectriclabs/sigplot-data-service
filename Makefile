all:
	go build

docker: all
	docker build -t sds:0.1 .
