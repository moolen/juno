SHELL=/bin/bash -o pipefail
DEST_DIR=/dist
LINUX_HEADERS=$(shell dnf list kernel-devel | awk '/^kernel-devel\..*/{print "/usr/src/kernels/"$$2".x86_64"}')

build:
	clang -D__KERNEL__ -D__ASM_SYSREG_H \
		-Wno-unused-value -Wno-pointer-sign -Wno-compare-distinct-pointer-types \
		-O2 -emit-llvm -c tcptracer-sock-bpf.c \
		-I . \
		$(foreach path,$(LINUX_HEADERS), -I $(path)/arch/x86/include -I $(path)/arch/x86/include/generated -I $(path)/include -I $(path)/include/generated/uapi -I $(path)/arch/x86/include/uapi -I $(path)/include/uapi) \
		-o - | llc -march=bpf -filetype=obj -o "$(DEST_DIR)/tcptracer-sock-ebpf.o"

	go-bindata -pkg tracer -prefix "$(DEST_DIR)/" -modtime 1 -o "$(DEST_DIR)/bindata.go" "$(DEST_DIR)/tcptracer-sock-ebpf.o"
