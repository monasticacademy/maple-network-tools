
DOCKER := docker --context=synology
EXTERNAL_PORT := 19875   # port to publish the container on
INTERNAL_PORT := 8000    # port for the Go binary to listen on

PROJECT := maple-network-health  # for the bigquery dataset
DATASET := network
TABLE := reachability
SCHEMA := timestamp:timestamp,googlereachable:bool,googleerror:string,googlelatency:integer,routerreachable:bool,routererror:string,routerlatency:integer,modemreachable:bool,modemerror:string,modemlatency:integer

# Compilation operations

.bin: *.go
	CGO_ENABLED=0 go build -o .bin

# Docker operations

image: .bin
	$(DOCKER) build . -t uplink-monitor

create:
	$(DOCKER) service create \
		--name uplink-monitor \
		--publish $(strip $(EXTERNAL_PORT)):$(strip $(INTERNAL_PORT)) \
		uplink-monitor

destroy:
	$(DOCKER) service rm uplink-monitor

inspect:
	$(DOCKER) service inspect --format pretty uplink-monitor

deploy:
	$(DOCKER) service update --force uplink-monitor

logs:
	$(DOCKER) service logs -f uplink-monitor

logtail:
	$(DOCKER) service logs --tail 20 uplink-monitor

gc:
	$(DOCKER) system prune

# Bigquery operations

create-dataset:
	bq --project_id $(PROJECT) mk $(DATASET)

create-table:
	bq --project_id $(PROJECT) mk -t $(DATASET).$(TABLE) $(SCHEMA)

update-table:
	bq --project_id $(PROJECT) update -t $(DATASET).$(TABLE) $(SCHEMA)

head:
	bq --project_id $(PROJECT) head $(DATASET).$(TABLE)

# Secret encryption and decryption

encrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --encrypt $(shell ls secrets/* | grep -v encrypted$$)

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --decrypt secrets/*.encrypted
