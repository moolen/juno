FROM docker.io/golang:1.14.1-buster

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y --no-install-recommends \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
		apt-utils \
		binutils \
		ca-certificates \
		clang-7 \
		coreutils \
		curl \
		gcc \
		git \
		iproute2 \
		libc6-dev \
		libc6-dev-i386 \
		libelf-dev \
		llvm-7 \
		m4 \
		make \
		pkg-config \
		python \
		rsync \
		unzip \
		wget \
		zip \
		zlib1g-dev \
		go-bindata \
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
	&& update-alternatives --install /usr/bin/clang clang /usr/bin/clang-7 100 \
	&& update-alternatives --install /usr/bin/llc llc /usr/bin/llc-7 100


RUN groupadd -r bpf && useradd -u 1000 -r -g bpf bpf
RUN mkdir -p /src /go
RUN chown bpf:bpf /src /go

USER bpf
