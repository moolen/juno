FROM golang:1.13.4-alpine3.10 as builder

WORKDIR /workspace

RUN sh -c "echo http://dl-cdn.alpinelinux.org/alpine/edge/community >> /etc/apk/repositories" && \
        apk update && \
        apk add \
        bcc gcc \
        libc-dev build-base \
        linux-headers curl

COPY go.mod go.mod
COPY go.sum go.sum
COPY Makefile Makefile
COPY pkg/ pkg/
COPY bpf/ bpf/
COPY cmd/ cmd/
COPY proto/ proto/
COPY vendor/ vendor/
COPY main.go main.go

RUN make binary

FROM alpine:3.10
WORKDIR /
RUN apk add --update ca-certificates graphviz
COPY --from=builder /workspace/bin/juno .
ADD entrypoint.sh /entrypoint.sh
ENTRYPOINT [ "/entrypoint.sh" ]
