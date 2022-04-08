DOCKER := docker --context=synology		# docker command prefix
#DOCKER := docker						# docker command prefix
IMAGE := thankyou-letter-printer		# docker image
SERVICE := thankyou-letter-printer		# docker swarm service name
PRINTER := none # http://brother-letterhead.maple.cml.me   # URI for the physical printer
PROJECT := thankyou-letters
TOPIC := print-queue
SUBSCRIPTION := print-queue

# hook for the cloudfunction
ENDPOINT := https://us-central1-thankyou-letters.cloudfunctions.net/print-google-doc

# ID of a Google doc used for testing
TEST_DOCUMENT := 1WSqBGHb-YJbtlx1qamLyuueU8A6vayXVprTQKtYBJ9k
#in team drive: TEST_DOCUMENT := 16jsBg1n4bOscEJLp5HPrCHDdjPU9ylUSP2mv6MpRYEI

default: .bin/thankyou-letter-printer

# Compilation operations

.bin/thankyou-letter-printer: *.go
	CGO_ENABLED=0 GOOS=linux go build -o .bin/thankyou-letter-printer

# Test operations

publish-test-message:
	curl $(strip $(ENDPOINT))?document=$(TEST_DOCUMENT)

# Docker operations

image: .bin/thankyou-letter-printer
	$(DOCKER) build . -t $(IMAGE)

testpdf:
	$(DOCKER) run -it --rm $(IMAGE) \
	./thankyou-letter-printer \
		--testpdf testdata/testfile.pdf \
		--printer $(PRINTER)

create:
	$(DOCKER) service create \
		--name $(SERVICE) \
		--env PRINTER=dir:/tmp \
		--env SUBSCRIPTION=$(SUBSCRIPTION) \
		$(IMAGE)

destroy:
	$(DOCKER) service rm $(SERVICE)

inspect:
	$(DOCKER) service inspect --format pretty $(SERVICE)

deploy:
	$(DOCKER) service update --force $(SERVICE)

logs:
	$(DOCKER) service logs -f $(SERVICE)

logtail:
	$(DOCKER) service logs --tail 20 $(SERVICE)

gc:
	$(DOCKER) system prune

run:
	# only for debugging
	$(DOCKER) run -it --rm -e PRINTER=dir:/tmp -e SUBSCRIPTION=$(SUBSCRIPTION) $(IMAGE)

# Pub/sub operations

create-topic:
	gcloud pubsub topics create print-queue \
		--project $(PROJECT)

create-subscription:
	gcloud pubsub subscriptions create $(SUBSCRIPTION) \
		--project $(PROJECT) \
		--topic $(TOPIC)

# Secret encryption and decryption

encrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --encrypt secrets/*.json

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --decrypt secrets/*.encrypted
