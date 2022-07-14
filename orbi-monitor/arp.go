package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

type arpEntry struct {
	IPAddr       net.IP
	HardwareType string
	Flags        string
	HardwareAddr net.HardwareAddr
	Mask         string
	Device       string
}

type arpTable struct {
	entries []arpEntry
}

// adapted from https://github.com/mostlygeek/arp
func loadARPTable() (*arpTable, error) {
	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	s.Scan() // skip the field descriptions

	var table arpTable
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if len(fields) != 6 {
			return nil, fmt.Errorf("expected 6 fields but got %d: %q", len(fields), strings.TrimSpace(line))
		}

		ip := net.ParseIP(fields[0])
		if ip == nil {
			return nil, fmt.Errorf("error parsing IP address: %q", fields[0])
		}

		hw, err := net.ParseMAC(fields[3])
		if err != nil {
			return nil, fmt.Errorf("error parsing MAC address: %w", err)
		}

		table.entries = append(table.entries, arpEntry{
			IPAddr:       ip,
			HardwareType: fields[1],
			Flags:        fields[2],
			HardwareAddr: hw,
			Mask:         fields[4],
			Device:       fields[5],
		})
	}

	return &table, nil
}
