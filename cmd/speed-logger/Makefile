
DOCKER := docker --context=nas

binary:
	CGO_ENABLED=0 go build

usb:
	CGO_ENABLED=0 go build
	cp speed-logger /mnt/chromeos/removable/CHINCHILLA/

image:
	$(DOCKER) build . -t speed-logger

start:
	$(DOCKER) rm -f speed-logger
	$(DOCKER) run -d --name=speed-logger speed-logger

stop:
	$(DOCKER) kill speed-logger

logtail:
	$(DOCKER) logs --tail 20 speed-logger

gc:
	$(DOCKER) system prune
