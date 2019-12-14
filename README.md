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

## Limitations

* this supports only a fixed number of CPUs (currently 2) see `tcptracker-sock-bpf.c` / `MAX_CPU`

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

# test server locally
$ kubectl port-forward svc/juno 3000:3000
$ ./bin/juno server
INFO[0002] received trace: trace:<time:<seconds:29 > IP:<source:"172.17.0.1" destination:"172.17.0.3" ipVersion:IPv4 > l4:<TCP:<source_port:35252 destination_port:8181 flags:<PSH:true ACK:true > > > l7:<http:<method:"GET" url:"/ready" protocol:"HTTP/1.1" > > >
INFO[0000] received trace: trace:<time:<seconds:22 > IP:<source:"172.17.0.1" destination:"172.17.0.2" ipVersion:IPv4 > l4:<TCP:<source_port:50774 destination_port:8080 flags:<PSH:true ACK:true > > > l7:<http:<method:"GET" url:"/health" protocol:"HTTP/1.1" > > >

# install demo app
$ kubectl apply -f ./hack/microservices-demo.yaml


```
