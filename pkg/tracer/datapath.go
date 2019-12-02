package tracer

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func compileAndLoad() (*ebpf.Collection, error) {
	buf, err := Asset("tcptracer-sock-ebpf.o")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't find asset")
	}
	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(buf))
	if err != nil {
		return nil, errors.Wrap(err, "error loading collection spec")
	}
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, errors.Wrap(err, "error creating collection")
	}
	return coll, nil
}

func replaceDatapath(coll *ebpf.Collection, ifacePrefix string) error {
	links, err := netlink.LinkList()
	if err != nil {
		return errors.Wrap(err, "error loading link list")
	}

	for _, link := range links {
		attrs := link.Attrs()
		matched, err := regexp.MatchString(ifacePrefix, attrs.Name)
		if err != nil {
			return errors.Wrapf(err, "error matching iface prefix: %s", ifacePrefix)
		}
		if !matched {
			log.Debugf("skipping link: %s", attrs.Name)
			continue
		}
		err = replaceQdisc(link)
		if err != nil {
			log.Errorf("error creating qdisc for %s: %s", attrs.Name, err)
			continue
		}
		log.Infof("created qdisc for: %s", attrs.Name)
		prog := coll.Programs["ingress"]
		if prog == nil {
			return fmt.Errorf("ingress program is missing")
		}
		err = createFilter(
			prog,
			link,
			netlink.HANDLE_MIN_EGRESS,
		)
		if err != nil {
			log.Errorf("error creating qdisc filter for %s: %s", attrs.Name, err.Error())
		}
	}
	return nil
}

func resetDatapath(coll *ebpf.Collection, ifacePrefix string) error {
	var errs []string
	links, err := netlink.LinkList()
	if err != nil {
		return err
	}
	for _, link := range links {
		attrs := link.Attrs()
		matched, err := regexp.MatchString(ifacePrefix, attrs.Name)
		if err != nil {
			return err
		}
		if !matched {
			log.Debugf("cleanup: skipping %s", attrs.Name)
			continue
		}
		err = deleteQdisc(link)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ", "))
	}
	return nil
}
