FROM fedora:24

ENV GOPATH /go

# vim-common is needed for xxd
# vim-minimal needs to be updated first to avoid an RPM conflict on man1/vim.1.gz
RUN dnf update -y vim-minimal && \
	dnf install -y llvm clang kernel-devel make binutils vim-common golang go-bindata

RUN groupadd -r bpf && useradd -u 1000 -r -g bpf bpf
RUN mkdir -p /src /go
RUN chown bpf:bpf /src /go

USER bpf
