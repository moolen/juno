
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


all: agent

# Run tests
test: fmt vet manifests misspell
	go test ./... -coverprofile cover.out

.PHONY: docs
docs: bin/hugo
	cd docs_src; ../$(HUGO) --theme book --destination ../docs

docs-live: bin/hugo
	cd docs_src; ../$(HUGO) server --minify --theme book

.PHONY: includes
includes:
	-mkdir -p vendor/github.com/iovisor/gobpf/elf/include
	-mkdir -p vendor/github.com/iovisor/gobpf/elf/lib
	curl -s $(GOBPF_ELF_SRC)/include/bpf.h -o $(GOBPF_ELF)/include/bpf.h
	curl -s $(GOBPF_ELF_SRC)/include/bpf_map.h -o $(GOBPF_ELF)/include/bpf_map.h
	curl -s $(GOBPF_ELF_SRC)/include/libbpf.h -o $(GOBPF_ELF)/include/libbpf.h
	curl -s $(GOBPF_ELF_SRC)/include/nlattr.h -o $(GOBPF_ELF)/include/nlattr.h
	curl -s $(GOBPF_ELF_SRC)/lib/netlink.c -o $(GOBPF_ELF)/lib/netlink.c
	curl -s $(GOBPF_ELF_SRC)/lib/nlattr.c -o $(GOBPF_ELF)/lib/nlattr.c

# Build agent binary
agent: includes # todo: add vet/fmt
	GOOS=linux GOARCH=amd64 GO111MODULE=on go build -mod vendor -a -o bin/juno main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet manifests
	mkdir -p bin/data
	go run ./main.go agent --store=bin/data

# Install CRDs into a cluster
install: kubectl-bin manifests
	kustomize build config/crd | $(KUBECTL) apply -f -

build-ebpf-object:
	docker build -f bpf/Dockerfile -t bpfbuilder .
	docker run --rm -it \
		-v $(PWD):/src \
		-v $(PWD)/bin:/dist/ \
		--workdir=/src/bpf \
		bpfbuilder \
		make -f ebpf.mk build

	sudo chown -R $(UID):$(UID) bin
	cp bin/tcptracer-ebpf.go pkg/tracer/tcptracer-ebpf.go


# Deploy agent in the configured Kubernetes cluster in ~/.kube/config
deploy: kubectl-bin manifests
	cd config/agent && kustomize edit set image agent=${IMG}
	kustomize build config/default | $(KUBECTL) apply -f -

# Checks if generated files differ
check-gen-files: docs quick-install
	git diff --exit-code

# Generate manifests e.g. CRD, RBAC etc.
manifests: bin/controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=harbor-sync paths="./..." output:crd:artifacts:config=config/crd/bases

quick-install: bin/kubectl
	$(KUBECTL) kustomize config/default/ > install/kubernetes/quick-install.yaml

misspell: bin/misspell
	$(MISSPELL) \
		-locale US \
		-error \
		api/* pkg/* docs_src/content/* config/* hack/* README.md CONTRIBUTING.md

# Run go fmt against code
fmt:
	go fmt ./pkg/...
	go fmt ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/...
	go vet ./cmd/...

# Run tests in container
docker-test:
	rm -rf bin
	docker build -t test:latest -f Dockerfile.test .
	docker run test:latest

docker-build:
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
