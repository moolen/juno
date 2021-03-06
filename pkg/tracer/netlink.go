package tracer

import (
	"fmt"
	"syscall"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func qdiscAttrs(link netlink.Link) *netlink.GenericQdisc {
	return &netlink.GenericQdisc{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: link.Attrs().Index,
			Handle:    netlink.MakeHandle(0xffff, 0),
			Parent:    netlink.HANDLE_CLSACT,
		},
		QdiscType: "clsact",
	}
}

func replaceQdisc(link netlink.Link) error {
	qdisc := qdiscAttrs(link)
	netlink.QdiscDel(qdisc)
	if err := netlink.QdiscAdd(qdisc); err != nil {
		return fmt.Errorf("netlink: replacing qdisc for %s failed: %s", link.Attrs().Name, err)
	}
	log.Printf("netlink: replacing qdisc for %s succeeded\n", link.Attrs().Name)
	return nil
}

func deleteQdisc(link netlink.Link) error {
	qdisc := qdiscAttrs(link)
	return netlink.QdiscDel(qdisc)
}

func filterAttrs(prog *ebpf.Program, link netlink.Link, parent uint32) *netlink.U32 {
	return &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: link.Attrs().Index,
			Parent:    parent,
			Handle:    netlink.MakeHandle(0, 1),
			Priority:  1,
			Protocol:  syscall.ETH_P_ALL,
		},
		ClassId: netlink.MakeHandle(1, 1),
		Actions: []netlink.Action{
			&netlink.BpfAction{
				Fd:   prog.FD(),
				Name: prog.String(),
			},
		},
	}
}

func createFilter(prog *ebpf.Program, link netlink.Link, parent uint32) error {
	filter := filterAttrs(prog, link, parent)
	err := netlink.FilterAdd(filter)
	if err != nil {
		return fmt.Errorf("failed to add filter: %s", err)
	}
	log.Printf("successfully added filter for %s \n", prog.String())
	return nil
}

func deleteFilter(prog *ebpf.Program, link netlink.Link, parent uint32) error {
	filter := filterAttrs(prog, link, parent)
	return netlink.FilterDel(filter)
}
