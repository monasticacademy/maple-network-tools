package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/alexflint/go-arg"
)

type DHCPLease struct {
	IP       string
	MAC      string
	Hostname string
}

type Traffic struct {
	From    string
	To      string
	Packets int
	Bytes   int
}

func hasPrefix(token, key string) (value string, ok bool) {
	if strings.HasPrefix(token, key) {
		return strings.TrimPrefix(token, key), true
	}
	return "", false
}

func main() {
	var args struct {
		Traffic string `arg:"positional,required"`
		DHCP    string `arg:"positional,required"`
	}
	arg.MustParse(&args)

	dhcpBuf, err := ioutil.ReadFile(args.DHCP)
	if err != nil {
		log.Fatal(err)
	}

	var leases []DHCPLease
	for _, line := range strings.Split(string(dhcpBuf), "\n") {
		var item DHCPLease
		for _, tok := range strings.Split(line, " ") {
			if v, ok := hasPrefix(tok, "address="); ok {
				item.IP = v
			}
			if v, ok := hasPrefix(tok, "mac-address="); ok {
				item.MAC = v
			}
			if v, ok := hasPrefix(tok, "host-name="); ok {
				item.Hostname = v
			}
		}
		var zero DHCPLease
		if item != zero {
			leases = append(leases, item)
		}
	}

	trafficBuf, err := ioutil.ReadFile(args.Traffic)
	if err != nil {
		log.Fatal(err)
	}

	var traffic []Traffic
	for _, line := range strings.Split(string(trafficBuf), "\n") {
		var item Traffic
		for _, tok := range strings.Split(line, " ") {
			if v, ok := hasPrefix(tok, "src-address="); ok {
				item.From = v
			}
			if v, ok := hasPrefix(tok, "dst-address="); ok {
				item.To = v
			}
			if v, ok := hasPrefix(tok, "packets="); ok {
				item.Packets, err = strconv.Atoi(v)
				if err != nil {
					log.Println(err)
				}
			}
			if v, ok := hasPrefix(tok, "bytes="); ok {
				item.Bytes, err = strconv.Atoi(v)
				if err != nil {
					log.Println(err)
				}
			}
		}
		var zero Traffic
		if item != zero {
			traffic = append(traffic, item)
		}
	}

	// match them up
	hostnameByIP := make(map[string]string)
	for _, lease := range leases {
		hostnameByIP[lease.IP] = lease.Hostname
	}

	type Usage struct {
		Hostname string
		Packets  int
		Bytes    int
	}
	usageByHostname := make(map[string]*Usage)
	for _, row := range traffic {
		var localIP string
		if strings.HasPrefix(row.From, "192.168.88.") {
			localIP = row.From
		} else if strings.HasPrefix(row.To, "192.168.88.") {
			localIP = row.To
		} else {
			continue
		}

		hostname, ok := hostnameByIP[localIP]
		if !ok {
			hostname = fmt.Sprintf("%s (not in dhcp)", localIP)
		}
		if hostname == "" {
			hostname = fmt.Sprintf("%s (missing hostname)", localIP)
		}

		usage := usageByHostname[hostname]
		if usage == nil {
			usage = &Usage{Hostname: hostname}
			usageByHostname[hostname] = usage
		}
		usage.Bytes += row.Bytes
		usage.Packets += row.Packets
	}

	for _, usage := range usageByHostname {
		fmt.Printf("%40s %10d bytes %10d packets\n", usage.Hostname, usage.Bytes, usage.Packets)
	}
}
