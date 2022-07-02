
# pull configuration from router to this repository (and convert line endings!)
pull:
	ssh microtik /export file=router.rsc
	scp microtik:router.rsc /tmp/router_crlf.rsc
	sed 's/\r$$//' < /tmp/router_crlf.rsc > router.rsc

# pull configuration from router to this repository (and convert line endings!)
pull-verbose:
	ssh microtik /export verbose file=router_verbose.rsc
	scp microtik:router_verbose.rsc /tmp/router_verbose_crlf.rsc
	sed 's/\r$$//' < /tmp/router_verbose_crlf.rsc > router_verbose.rsc

# compare the live configuration of the router to router.rsc
diff:
	ssh microtik /export file=live.rsc
	scp microtik:live.rsc /tmp/live_crlf.rsc
	sed 's/\r$$//' < /tmp/live_crlf.rsc > /tmp/live.rsc
	diff router.rsc /tmp/live.rsc

# does not really work yet
push:
	echo "not implemented"
#	scp router.rsc microtik:
#	ssh microtik /import file=router.rsc


# Test packets

# send a UDP packet from 192.168.88.5 to google
sendpacket:
	sudo sendip -p ipv4 -is 192.168.88.5 -p udp -us 19870 -ud 19871 google.com

# print the contents of the packetlog
packetlog:
	scp microtik:packetlog.0.txt /tmp/packetlog
	cat /tmp/packetlog
	wc /tmp/packetlog

routelog:
	scp microtik:routelog.0.txt /tmp/routelog
	cat /tmp/routelog
	wc /tmp/routelog

# Secret encryption and decryption

encrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../cmd/crypt/*.go --encrypt $(shell ls secrets/* | grep -v '\.encrypted')

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../cmd/crypt/*.go --decrypt secrets/*.encrypted
