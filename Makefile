build:
	go build -o bin/metalcore cmd/metalcore/metalcore.go

run:
	docker run -it --rm --network="host" metalcore-test

compile:
	echo "Compiling for every OS and platform:"
	GOOS=linux GOARCH=amd64 go build -o bin/metalcore cmd/metalcore/metalcore.go
	GOOS=windows GOARCH=amd64 go build -o bin/metalcore.exe cmd/metalcore/metalcore.go
	#GOOS=linux GOARCH=arm64 go build -o bin/metalcore-imp-arm64 src/imp.go
	#GOOS=linux GOARCH=arm go build -o bin/metalcore-imp-arm src/imp.go
	GOOS=linux GOARCH=amd64 go build -o bin/service cmd/service/service.go
	GOOS=windows GOARCH=amd64 go build -o bin/service.exe cmd/service/service.go
	GOOS=linux GOARCH=amd64 go build -o bin/clientapp cmd/clientapp/clientapp.go
	GOOS=windows GOARCH=amd64 go build -o bin/clientapp.exe cmd/clientapp/clientapp.go

image:
	docker build -t metalcore-test -f build/docker/Dockerfile .

container:
	docker run --rm --network=host --cpus=2 --env LOGLEVEL=trace --env LOGPRETTYPRINT=on -it metalcore-test

containerdaemon:
	echo "starting a new container with a random name..."
	docker run -d --rm --network="host" metalcore-test
	docker ps -l

# ugly way to create a bunch of containers without k8s/k3s/swarm
NUMBERS := 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63 64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 86 87 88 89 90 91 92 93 94 95 96 97 98 99 100
startfakecluster:
	$(foreach var,$(NUMBERS),docker run -d --rm --network="host" --name metalcore-worker-$(var) metalcore-test;)
	docker ps

stopfakecluster:
	docker ps --format '{{.Names}}' | grep "^metalcore-worker-" | awk '{print $1}' | xargs -I {} docker stop {}
	docker ps

all: compile image