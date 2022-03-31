DOCKER := docker --context=synology		# docker command prefix
IMAGE := thankyou-letter-printer		# docker image
SERVICE := thankyou-letter-printer		# docker swarm service name
PRINTER := http://brother-letterhead.maple.cml.me   # URI for the physical printer
ENDPOINT := https://us-central1-thankyou-letters.cloudfunctions.net/print-google-doc # hook for the cloudfunction

# ID of a Google doc used for testing
TEST_DOCUMENT := 123

default: .bin/thankyou-letter-printer

# Compilation operations

.bin/thankyou-letter-printer: *.go
	CGO_ENABLED=0 GOOS=linux go build -o .bin/thankyou-letter-printer

# Test operations

publish-test:
	curl $(ENDPOINT)?document=$(TEST_DOCUMENT)

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

# Secret encryption and decryption

encrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --encrypt secrets/*.json

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --decrypt secrets/*.encrypted
