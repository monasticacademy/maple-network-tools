
# pull configuration from router to this repository
pull:
	ssh microtik /export file=router.rsc
	scp microtik:router.rsc .

# Compare the live configuration of the router to router.rsc
diff:
	ssh microtik /export file=live.rsc
	scp microtik:live.rsc /tmp/live.rsc
	diff router.rsc /tmp/live.rsc


# does not really work yet
push:
	echo "not implemented"
#	scp router.rsc microtik:
#	ssh microtik /import file=router.rsc

# Secret encryption and decryption

encrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../cmd/crypt/*.go --encrypt $(shell ls secrets/* | grep -v '\.encrypted')

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../cmd/crypt/*.go --decrypt secrets/*.encrypted
