
version ?= test
IMAGE_REPO = quay.io/moolen/juno
IMG ?= ${IMAGE_REPO}:${version}
CRD_OPTIONS ?= "crd:trivialVersions=true"

GOPATH=$(shell go env GOPATH)
HUGO=bin/hugo
KUBECTL=bin/kubectl
MISSPELL=bin/misspell

UID=$(shell id -u)
PWD=$(shell pwd)

GOBPF_ELF = vendor/github.com/iovisor/gobpf/elf
GOBPF_ELF_SRC = https://raw.githubusercontent.com/iovisor/gobpf/master/elf

all: proto build-ebpf binary docker-build

.PHONY: clean
clean:
	rm -rf bin/ vendor/

test: fmt vet misspell
	go test ./... -coverprofile cover.out

binary: vendor
	GOOS=linux GOARCH=amd64 GO111MODULE=on go build -mod vendor -a -o bin/juno main.go

.PHONY: proto
proto:
	protoc -I proto proto/tracer.proto --go_out=plugins=grpc:./proto

vendor:
	go mod vendor

install:
	kustomize build config/default | $(KUBECTL) apply -f -

build-ebpf: bin
	docker build -f bpf/Dockerfile -t bpfbuilder .
	docker run --rm -it \
		-v $(PWD):/src \
		-v $(PWD)/bin:/dist/ \
		--workdir=/src/bpf \
		bpfbuilder \
		make -f ebpf.mk build

	cp bin/bindata.go pkg/tracer/bindata.go

deploy:
	cd config/agent && kustomize edit set image agent=${IMG}
	kustomize build config/default | $(KUBECTL) apply -f -

misspell: bin/misspell
	$(MISSPELL) \
		-locale US \
		-error \
		api/* pkg/* docs_src/content/* config/* hack/* README.md CONTRIBUTING.md

fmt:
	go fmt ./pkg/...
	go fmt ./cmd/...

vet:
	go vet ./pkg/...
	go vet ./cmd/...

docker-test:
	rm -rf bin
	docker build -t test:latest -f Dockerfile.test .
	docker run test:latest

docker-build: proto build-ebpf
	docker build . -t ${IMG}

docker-push:
	docker push ${IMG}

docker-push-latest:
	docker tag ${IMG} ${IMAGE_REPO}:latest
	docker push ${IMAGE_REPO}:latest

docker-release: docker-build docker-push docker-push-latest

release: quick-install agent docker-release
	tar cvzf bin/juno.tar.gz bin/juno

bin/misspell:
	curl -sL https://github.com/client9/misspell/releases/download/v0.3.4/misspell_0.3.4_linux_64bit.tar.gz | tar -xz -C /tmp/
	mkdir bin; cp /tmp/misspell bin/misspell

bin/hugo:
	curl -sL https://github.com/gohugoio/hugo/releases/download/v0.57.2/hugo_extended_0.57.2_Linux-64bit.tar.gz | tar -xz -C /tmp/
	mkdir bin; cp /tmp/hugo bin/hugo

bin/kubectl:
	curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.15.0/bin/linux/amd64/kubectl
	chmod +x ./kubectl
	mkdir bin; mv kubectl bin/kubectl

bin:
	mkdir bin
