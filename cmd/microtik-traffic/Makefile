
DOCKER := docker --context=nas

# TODO: switch to docker secrets instead of this
MICROTIK_PASSWORD := $(shell cat secrets/microtik-password)

microtik-traffic: *.go
	CGO_ENABLED=0 go build

image: microtik-traffic
	$(DOCKER) build . -t microtik-traffic

restart:
	-$(DOCKER) rm -f microtik-traffic
	$(DOCKER) run \
		-d \
		--name=microtik-traffic \
		-e PASS=$(MICROTIK_PASSWORD) \
		microtik-traffic

stop:
	$(DOCKER) kill microtik-traffic

logtail:
	$(DOCKER) logs --tail 20 microtik-traffic

logfollow:
	$(DOCKER) logs -f microtik-traffic

gc:
	$(DOCKER) system prune
