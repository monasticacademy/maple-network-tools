
DOCKER := docker --context=nas

microtik-traffic: *.go
	CGO_ENABLED=0 go build

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
