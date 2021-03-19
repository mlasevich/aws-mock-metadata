VERSION_VAR := main.VERSION
REPO_VERSION := $(shell git describe --always --dirty --tags)
GOBUILD_VERSION_ARGS := -ldflags "-X $(VERSION_VAR)=$(REPO_VERSION)"
GIT_HASH := $(shell git rev-parse --short HEAD)
APP_NAME := aws-mock-metadata

include .env

setup:
	go get -v
	go get -v -u github.com/githubnemo/CompileDaemon
	go get -v -u github.com/alecthomas/gometalinter
	gometalinter --install --update

build: *.go
	gofmt -w=true .
	go build -o bin/$(APP_NAME) $(GOBUILD_VERSION_ARGS) github.com/jtblin/$(APP_NAME)

test: build
	go test

junit-test: build
	go get github.com/jstemmer/go-junit-report
	go test -v | go-junit-report > test-report.xml

check: build
	gometalinter ./...

watch:
	CompileDaemon -color=true -build "make test"

commit-hook:
	cp dev/commit-hook.sh .git/hooks/pre-commit

build-linux:
	gofmt -w=true .
	GOOS=linux GOARCH=amd64 go build -o bin/$(APP_NAME)-linux $(GOBUILD_VERSION_ARGS) github.com/jtblin/$(APP_NAME)

cross:
	 CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o bin/$(APP_NAME)-linux .

docker: cross
	 docker build -t jtblin/$(APP_NAME):$(GIT_HASH) .

version:
	@echo $(REPO_VERSION)

run:
	@AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		AWS_SESSION_TOKEN=$(AWS_SESSION_TOKEN) bin/$(APP_NAME) --availability-zone=$(AVAILABILITY_ZONE) \
		--instance-id=$(INSTANCE_ID) --hostname=$(HOSTNAME) --role-name=$(ROLE_NAME) --role-arn=$(ROLE_ARN) \
		--app-port=$(APP_PORT)

run-macos:
	bin/server-macos

run-linux:
	bin/server-linux

run-docker:
	@docker run -it --rm -p 80:$(APP_PORT) -e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) -e AWS_SESSION_TOKEN=$(AWS_SESSION_TOKEN) \
		jtblin/aws-mock-metadata:$(GIT_HASH) --availability-zone=$(AVAILABILITY_ZONE) --instance-id=$(INSTANCE_ID) \
		--hostname=$(HOSTNAME) --role-name=$(ROLE_NAME) --role-arn=$(ROLE_ARN) --app-port=$(APP_PORT) \
		--vpc-id=$(VPC_ID) --private-ip=$(PRIVATE_IP)

clean:
	rm -f bin/$(APP_NAME)*
	docker rm $(shell docker ps -a -f 'status=exited' -q) || true
	docker rmi $(shell docker images -f 'dangling=true' -q) || true

release: docker
	docker push jtblin/$(APP_NAME):$(GIT_HASH)
	docker tag -f jtblin/$(APP_NAME):$(GIT_HASH) jtblin/$(APP_NAME):latest
	docker push jtblin/$(APP_NAME):latest
