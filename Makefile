.DEFAULT_GOAL := all

build:
	go build -o bin/metalcore cmd/metalcore/metalcore.go

compile:
	echo "Compiling for every OS and platform:"
	GOOS=linux GOARCH=amd64 go build -o bin/metalcore cmd/metalcore/metalcore.go
	#GOOS=windows GOARCH=amd64 go build -o bin/metalcore.exe cmd/metalcore/metalcore.go
	#GOOS=linux GOARCH=arm64 go build -o bin/metalcore-imp-arm64 src/imp.go
	#GOOS=linux GOARCH=arm go build -o bin/metalcore-imp-arm src/imp.go
	GOOS=linux GOARCH=amd64 go build -o bin/service cmd/service/service.go
	#GOOS=windows GOARCH=amd64 go build -o bin/service.exe cmd/service/service.go
	GOOS=linux GOARCH=amd64 go build -o bin/clientapp cmd/clientapp/clientapp.go
	#GOOS=windows GOARCH=amd64 go build -o bin/clientapp.exe cmd/clientapp/clientapp.go

images:
	docker-compose --file build/docker/docker-compose.yml build
	#docker build -t metalcore-worker -f build/docker/metalcore-worker/Dockerfile bin/
	#docker build -t metalcore-queue build/docker/metalcore-queue/
	docker image prune -f

runbareimp:
	QUEUEHOST=localhost QUEUEPORT=11300 TASKQUEUENAME=tasks RESULTQUEUENAME=results CONNSHAREFACTOR=1 LOGPRETTYPRINT=on SERVICEPATH=bin/service bin/metalcore

container:
	docker run --rm --network=host --env LOGPRETTYPRINT=on -it metalcore-worker

containerdebug:
	docker run --rm --network=host --env LOGLEVEL=trace --env LOGPRETTYPRINT=on -it metalcore-worker

containershell:
	docker run --rm --network=host --env LOGLEVEL=trace --env LOGPRETTYPRINT=on -it metalcore-worker /bin/sh

containerdaemon:
	echo "starting a new container with a random name..."
	docker run -d --rm --network="host" metalcore-worker
	docker ps -l

queue:
	docker run -d --rm -p=11300:11300 --name=metalcore-queue metalcore-queue -z 536870912

benchmark:
	bin/clientapp -tasks=100000 -size=0 -pretty -parallel=4

up:
	docker-compose --file build/docker/docker-compose.yml --project-name metalcore up --detach --scale worker=3

down:
	docker-compose --file build/docker/docker-compose.yml --project-name metalcore down --volumes

clean: down
	rm -f bin/*
	go mod tidy

all: clean compile images
