#!/bin/bash

MICROTIK=admin@192.168.88.1

ssh ${MICROTIK} /ip accounting snapshot take
ssh ${MICROTIK} /ip accounting snapshot print terse > traffic.txt
ssh ${MICROTIK} /ip dhcp-server lease print terse > dhcp.txt
