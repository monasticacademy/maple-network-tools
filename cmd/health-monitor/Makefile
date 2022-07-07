
DOCKER := docker --context=synology
EXTERNAL_PORT := 19870   # port to publish the container on
INTERNAL_PORT := 8000    # port for the Go binary to listen on

PROJECT := maple-network-health  # for the bigquery dataset
DATASET := network
TABLE := health
SCHEMA := timestamp:timestamp,operation:string,error:string,duration:integer

# Compilation operations

.bin: *.go
	CGO_ENABLED=0 go build -o .bin

# Docker operations

image: .bin
	$(DOCKER) build . -t health-monitor

create:
	$(DOCKER) service create \
		--name health-monitor \
		--publish $(strip $(EXTERNAL_PORT)):$(strip $(INTERNAL_PORT)) \
		health-monitor

destroy:
	$(DOCKER) service rm health-monitor

inspect:
	$(DOCKER) service inspect --format pretty health-monitor

deploy:
	$(DOCKER) service update --force health-monitor

logs:
	$(DOCKER) service logs -f health-monitor

logtail:
	$(DOCKER) service logs --tail 20 health-monitor

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
	go run ../crypt/*.go --encrypt secrets/*.json

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --decrypt secrets/*.encrypted
