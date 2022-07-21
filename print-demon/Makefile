DOCKER := docker --context=synology		# docker command prefix
#DOCKER := docker						# docker command prefix
IMAGE := print-demon		# docker image
SERVICE := print-demon		# docker swarm service name
PRINTER := http://192.168.88.133   # URI for the physical printer
PROJECT := thankyou-letters
TOPIC := print-queue
SUBSCRIPTION := print-queue

# hook for the cloudfunction
ENDPOINT := https://us-central1-thankyou-letters.cloudfunctions.net/print-google-doc

# ID of a Google doc used for testing
TEST_DOCUMENT := 1WSqBGHb-YJbtlx1qamLyuueU8A6vayXVprTQKtYBJ9k
#in team drive: TEST_DOCUMENT := 16jsBg1n4bOscEJLp5HPrCHDdjPU9ylUSP2mv6MpRYEI

default: .bin/print-demon

# Compilation operations

.bin/print-demon: *.go
	CGO_ENABLED=0 GOOS=linux go build -o .bin/print-demon

# Test operations

publish-test-message:
	curl $(strip $(ENDPOINT))?document=$(TEST_DOCUMENT)

pdf-to-pcl:
	gs -o testdata/thankyou.pcl \
    -sDEVICE=pxlcolor \
    -f testdata/thankyou.pdf

# Docker operations

image: .bin/print-demon
	$(DOCKER) build . -t $(IMAGE)

testpdf:
	$(DOCKER) run -it --rm $(IMAGE) \
		./print-demon \
		--testpdf testdata/testfile.pdf \
		--printer $(PRINTER)

create:
	$(DOCKER) service create \
		--name $(SERVICE) \
		--env PRINTER=$(PRINTER) \
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
	$(DOCKER) run \
		-it \
		--rm \
		-e PRINTER=dir:/tmp \
		-e SUBSCRIPTION=$(SUBSCRIPTION) \
		$(IMAGE)

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
	go run ../crypt/*.go --encrypt $(shell ls secrets/* | grep -v encrypted$$)

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --decrypt secrets/*.encrypted
