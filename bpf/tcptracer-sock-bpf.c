#define KBUILD_MODNAME "juno-socktracer"

#include <linux/kconfig.h>
#include <linux/ptrace.h>
#include <stddef.h>
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/if_packet.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/udp.h>
#include <linux/tcp.h>
#include <net/sock.h>

#include "common.h"
#include "bpf_helpers.h"

#define SAMPLE_SIZE 128u

// TODO: this requires re-compiling duing runtime (which we do not want to support)
// read man 7 bpf-helpers at bpf_perf_event_read_value
// clang offers a macro: __NR_CPUS__
// see: https://github.com/cilium/cilium/blob/e049cfa7e253d224f7fdfdc30390e62556c5e6ee/bpf/lib/events.h#L14
#define MAX_CPUS 2
#define ETH_HLEN 14

#ifndef __packed
#define __packed __attribute__((packed))
#endif

#define min(x, y) ((x) < (y) ? (x) : (y))

#define bpf_printk(fmt, ...)					\
({								\
           char ____fmt[] = fmt;				\
           bpf_trace_printk(____fmt, sizeof(____fmt),	\
                ##__VA_ARGS__);			\
})

/* Metadata will be in the perf event before the packet data. */
struct trace_metadata {
    __u32 ifindex;
    __u16 pkt_len;
} __packed;

struct bpf_map_def SEC("maps/EVENTS_MAP") EVENTS_MAP = {
    .type = BPF_MAP_TYPE_PERF_EVENT_ARRAY,
    .key_size = 0,
    .value_size = 0,
    .max_entries = MAX_CPUS,
};

static __always_inline void send_trace(struct __sk_buff *skb, __u64 sample_size) {

    uint64_t skb_len = (uint64_t)skb->len;

    struct trace_metadata metadata = {
        .ifindex = skb->ifindex,
        .pkt_len = skb_len,
    };

    bpf_printk("trace sample size: %llu\n", sample_size);
    int ret = bpf_perf_event_output(skb, &EVENTS_MAP,
            (sample_size << 32) | BPF_F_CURRENT_CPU,
            &metadata, sizeof(metadata));
    if (ret != 0) {
        bpf_printk("trace failed: %d\n", ret);
    }
}

SEC("action/ingress")
int ingress(struct __sk_buff *skb)
{
    bpf_printk("got packet: %d\n", skb->data_end);
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;

    struct hdr_cursor nh = { .pos = data };
    long eth_type, ip_type;
    struct ethhdr *eth;
    struct iphdr *ip;
    struct ipv6hdr *ip6;
    struct udphdr *udp;
    struct tcphdr *tcp;

    __u32 tcp_header_length = 0;
    __u32 ip_header_length = 0;
    __u32 payload_offset = 0;
    __u32 payload_length = 0;
    __u16 ip_len = 0;
    __u64 sample_size = 0;


    eth_type = parse_ethhdr(&nh, data_end, &eth);
    if (eth_type < 0) {
        bpf_printk("return invalid eth type: %d\n", eth_type);
        return TC_ACT_OK;
    }

    if (eth_type == bpf_htons(ETH_P_IP)) {
        ip_type = parse_iphdr(&nh, data_end, &ip);
        if (ip == NULL) {
            bpf_printk("ip is null: %d\n", ip);
            return TC_ACT_OK;
        }
        ip_header_length = ip->ihl << 2;
        ip_len = bpf_ntohs(ip->tot_len) >> 8;
    } else if (eth_type == bpf_htons(ETH_P_IPV6)) {
        // ip_type = parse_ip6hdr(&nh, data_end, &ip6);
        // if (ip6 == NULL) {
        //     bpf_printk("ip6 is null: %d\n", ip);
        //     return TC_ACT_OK;
        // }
        // ip_header_length = 40 << 2; // fixed header size
        //ip_len = bpf_ntohs(ip6->payload_len) >> 8;
    } else {
        bpf_printk("return eth type: %lu / %lu\n", eth_type, ETH_P_IP);
        return TC_ACT_OK;
    }

    if (ip_type == IPPROTO_UDP) {
        if (parse_udphdr(&nh, data_end, &udp) < 0) {
            bpf_printk("return udp hdr: %d\n", eth_type);
            return TC_ACT_OK;
        }
        if (udp == NULL) {
            bpf_printk("wtf udp is null: %d\n", eth_type);
            return TC_ACT_OK;
        }
        payload_offset = ETH_HLEN + ip_header_length;
        payload_length = (bpf_ntohs(udp->len)>>8); // udp->len = header + payload
        bpf_printk("udp len be: %d\n", udp->len);
        bpf_printk("payload_length: %lu\n", payload_length);
        bpf_printk("payload_offset: %lu\n", payload_offset);
        sample_size = min(payload_offset + payload_length, SAMPLE_SIZE);
        send_trace(skb, sample_size);
    } else if (ip_type == IPPROTO_TCP) {
        if (parse_tcphdr(&nh, data_end, &tcp) < 0) {
            bpf_printk("return tcp hdr: %d\n", eth_type);
            return TC_ACT_OK;
        }
        if (tcp == NULL) {
            bpf_printk("wtf tcp is null: %d\n", eth_type);
            return TC_ACT_OK;
        }
        tcp_header_length = tcp->doff << 2;
        payload_offset = ETH_HLEN + ip_header_length + tcp_header_length;
        payload_length = ip_len - ip_header_length - tcp_header_length;
        sample_size = min(payload_offset + payload_length, SAMPLE_SIZE);
        send_trace(skb, sample_size);
    }

    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
