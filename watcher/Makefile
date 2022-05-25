
APP_NAME = kyma-watcher
IMG_REPO := $(DOCKER_PUSH_REPOSITORY)$(DOCKER_PUSH_DIRECTORY)
IMG_NAME := $(IMG_REPO)/$(APP_NAME)
TAG := $(DOCKER_TAG)

FLAGS = -ldflags '-s -w'

.PHONY: resolve
resolve:
	go mod tidy -compat=1.18

.PHONY: build
build: build-linux build-darwin build-linux-arm build-windows

.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/kyma-watcher-linux $(FLAGS) ./

.PHONY: build-linux-arm
build-linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/kyma-watcher-arm $(FLAGS) ./

.PHONY: build-darwin
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/kyma-watcher-darwin $(FLAGS) ./

.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/kyma-watcher-windows.exe $(FLAGS) ./

.PHONY: docker-build
docker-build:
	docker build -t $(APP_NAME)/kyma-watcher:latest -f Dockerfile .

.PHONY: docker-push
docker-push:
ifdef IMG_REPO
	docker tag $(APP_NAME)/kyma-watcher $(IMG_NAME)/mothership:$(TAG)
	docker push $(IMG_NAME)/kyma-watcher:$(TAG)
endif

.PHONY: clean
clean:
	rm -rf bin
