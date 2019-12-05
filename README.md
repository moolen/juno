# Juno
Network tracing and auditing for Kubernetes.

## TODO

Poc #1
* [x] run kprobe for tcp connect/accept
* [x] draw dependency graph based on observed connections (see /tmp/graph.svg)

PoC #2
* [x] run eBPF program on veth to extract traffic flow information
* [x] implement auditing use-case: implement event buffer map per veth interface

* [ ] implement central component to collect traffic information

## Notes

* docker/moby does not support cgroup2

## Installation

```
kubectl apply -k config/default/
```

## Example

Preprequisites:
* have juno installed


follow hubble example:

```
kubectl create namespace jobs-demo
kubectl -n jobs-demo apply -f https://app.isovalent.com/demos/jobs.yaml
```

Once the pods are up generate some traffic:

```
curl -sLO https://app.isovalent.com/demos/jobs-traffic.sh && bash jobs-traffic.sh jobs-demo
```

## Development
```sh
$ minikube start
# build bpf bytecode and protobuf defs
$ make all

# build docker container in minikube
$ eval $(minikube docker-env)
$ docker build . -t quay.io/moolen/juno:test
$ kubectl apply -k config/default

# demo app
$ kubectl apply -f ./hack/microservices-demo.yaml
```
