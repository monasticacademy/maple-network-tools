PROJECT := scratch-12345
ZONE := us-central1-c
INSTANCE := maple-ridgewave-modem-syslogd
EXECUTABLE := syslog-udp-to-gcp
HOSTPORT := :19870
LOGNAME := ridgewave-syslog
COMMAND := bash -c 'sudo killall $(EXECUTABLE) > /dev/null; nohup ./$(EXECUTABLE) $(HOSTPORT) --project $(PROJECT) --logname $(LOGNAME) > $(EXECUTABLE).out 2> $(EXECUTABLE).err &'
#COMMAND := bash -c 'sudo killall $(EXECUTABLE); pwd; ls; ./$(EXECUTABLE); echo done'

$(EXECUTABLE): main.go
	go build

deploy: syslog-udp-to-gcp
	gcloud beta compute scp \
		--zone $(ZONE) \
		--project $(PROJECT) \
		$(EXECUTABLE) $(INSTANCE):

restart:
	gcloud beta compute ssh \
		--zone $(ZONE) \
		--project $(PROJECT) \
		$(INSTANCE) \
		--command "$(COMMAND)"

stop:
	gcloud beta compute ssh \
		--zone $(ZONE) \
		--project $(PROJECT) \
		$(INSTANCE) \
		--command "sudo killall $(EXECUTABLE)"

log:
	gcloud beta compute ssh \
		--zone $(ZONE) \
		--project $(PROJECT) \
		$(INSTANCE) \
		--command "cat $(EXECUTABLE).err"

tail:
	gcloud beta compute ssh \
		--zone $(ZONE) \
		--project $(PROJECT) \
		$(INSTANCE) \
		--command "tail -f $(EXECUTABLE).err"

status:
	gcloud beta compute ssh \
		--zone $(ZONE) \
		--project $(PROJECT) \
		$(INSTANCE) \
		--command "ps aux | grep $(EXECUTABLE) | grep -v grep"

ssh:
	gcloud beta compute ssh \
		--zone $(ZONE) \
		--project $(PROJECT) \
		$(INSTANCE)

ping:
	echo ping | nc -q 0 -u 104.154.222.5 19870
