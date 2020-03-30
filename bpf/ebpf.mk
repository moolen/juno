SHELL=/bin/bash -o pipefail
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
DEST_DIR=/dist

CLANG_FLAGS := -I ${ROOT_DIR} -I ${ROOT_DIR}/include -target bpf -emit-llvm -O2 -g
CLANG_FLAGS += -Wno-address-of-packed-member -Wno-unknown-warning-option -Wno-compare-distinct-pointer-types
LLC_FLAGS   := -march=bpf -mcpu=probe -mattr=dwarfris

CLANG  ?= clang
LLC    ?= llc
HOSTCC ?= gcc

build:
	$(CLANG) $(CLANG_FLAGS) -c tcptracer-sock-bpf.c -o - \
	| $(LLC) $(LLC_FLAGS) -filetype=obj -o "$(DEST_DIR)/tcptracer-sock-ebpf.o"
	go-bindata -pkg tracer -prefix "$(DEST_DIR)/" -modtime 1 -o "$(DEST_DIR)/bindata.go" "$(DEST_DIR)/tcptracer-sock-ebpf.o"
