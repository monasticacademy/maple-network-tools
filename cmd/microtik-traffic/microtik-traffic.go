package main

import (
	_ "embed"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"golang.org/x/crypto/ssh"
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

//go:embed microtik.pub
var microtikServerKey []byte

func main() {
	var args struct {
		Router   string `help:"Hostname or IP address of router"`
		User     string `help:"SSH username for router"`
		Pass     string `help:"SSH password for router"`
		Interval time.Duration
	}
	arg.MustParse(&args)

	// parse the embeded public key for our router
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(microtikServerKey)
	if err != nil {
		log.Fatal("error parsing server SSH key: ", err)
	}

	// open an ssh connection to the router
	sshClient, err := ssh.Dial("tcp", args.Router, &ssh.ClientConfig{
		User:            args.User,
		Auth:            []ssh.AuthMethod{ssh.Password(args.Pass)},
		HostKeyCallback: ssh.FixedHostKey(pubkey),
		//HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatal("error dialing ssh on router: ", err)
	}

	// create a session to fetch the dhcp lease table
	dhcpSession, err := sshClient.NewSession()
	if err != nil {
		log.Fatal("error opening SSH session: ", err)
	}
	defer dhcpSession.Close()

	// fetch the DHCP lease table
	dhcpBuf, err := dhcpSession.CombinedOutput("/ip dhcp-server lease print terse")
	if err != nil {
		log.Fatal("error running DHCP command: ", err)
	}

	// parse the dhcp lease table
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

	// create a session to fetch the traffic table
	trafficSession, err := sshClient.NewSession()
	if err != nil {
		log.Fatal("error opening SSH session: ", err)
	}
	defer dhcpSession.Close()

	// fetch the traffic table
	trafficBuf, err := trafficSession.CombinedOutput("/ip accounting snapshot print terse")
	if err != nil {
		log.Fatal("error running traffic command: ", err)
	}

	// parse the traffic table
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

	// calculate usage per hostname
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

	// print usage info
	for _, usage := range usageByHostname {
		fmt.Printf("%40s %10d bytes %10d packets\n", usage.Hostname, usage.Bytes, usage.Packets)
	}
}
