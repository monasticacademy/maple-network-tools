DOCKER := docker --context=nas

.bin:
	CGO_ENABLED=0 go build -o .bin

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

encrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --encrypt $(shell ls secrets/* | grep -v encrypted$$)

decrypt-secrets:
	echo "Please enter the password from bitwarden under 'maple network tools'..."
	go run ../crypt/*.go --decrypt secrets/*.encrypted
