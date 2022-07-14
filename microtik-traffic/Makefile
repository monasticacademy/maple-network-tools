
DOCKER := docker --context=synology
ROUTER := admin@microtik.maple.cml.me
TABLE := maple.bandwidth_usage
SCHEMA := begin:timestamp,duration:integer,host:string,mac:string,bytes:integer,packets:integer

# Compilation operations

microtik-traffic: *.go
	CGO_ENABLED=0 go build

dry-run:
	env $(shell cat secrets/secrets | xargs) go run *.go --testssh

# Docker operations

image: microtik-traffic
	$(DOCKER) build . -t microtik-traffic

create:
	$(DOCKER) service create \
		--name microtik-traffic \
		--env-file secrets/secrets \
		microtik-traffic

destroy:
	$(DOCKER) service rm microtik-traffic

inspect:
	$(DOCKER) service inspect --format pretty microtik-traffic

deploy:
	$(DOCKER) service update --force microtik-traffic

logs:
	$(DOCKER) service logs -f microtik-traffic

logtail:
	$(DOCKER) service logs --tail 20 microtik-traffic

gc:
	$(DOCKER) system prune

# Bigquery operations

create-table:
	bq mk -t $(TABLE) $(SCHEMA)

update-table:
	bq update -t $(TABLE) $(SCHEMA)

head:
	bq head $(TABLE)

# SSH operations

fetch-router-fingerprint:
	ssh-keyscan microtik.maple.cml.me > microtik.pub

ssh-key:
	@# create a read-only user account for the monitor tool
	ssh $(ROUTER) /user add name=traffic-monitor group=read password=<password>

	@# create a new key pair
	ssh-keygen -t rsa secrets/microtik-traffic.key

	@# copy the public key to the router
	scp secrets/microtik-traffic.key.pub $(ROUTER):key.pub

	@# add the key to the new user account
	ssh $(ROUTER) /user ssh-keys import public-key-file=key.pub user=microtik-traffic

# RouterOS operations

fetch-snapshot:
	ssh $(ROUTER) /ip accounting snapshot take
	ssh $(ROUTER) /ip accounting snapshot print terse > traffic.txt
	ssh $(ROUTER) /ip dhcp-server lease print terse > dhcp.txt

# Secret encryption and decryption

encrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --encrypt $(shell ls secrets/* | grep -v encrypted$$)

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --decrypt secrets/*.encrypted
