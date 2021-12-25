
DOCKER := docker --context=nas
MICROTIK_PASSWORD := $(shell cat secrets/microtik-password)

binary:
	CGO_ENABLED=0 go build

image:
	$(DOCKER) build . -t microtik-traffic

start:
	$(DOCKER) rm -f microtik-traffic
	$(DOCKER) run -d --name=microtik-traffic -e PASS=$(MICROTIK_PASSWORD) microtik-traffic

stop:
	$(DOCKER) kill microtik-traffic

logtail:
	$(DOCKER) logs --tail 20 microtik-traffic

gc:
	$(DOCKER) system prune
