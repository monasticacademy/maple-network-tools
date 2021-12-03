
# address where file-transfer utility is running on the orbi
ORBI := 10.0.0.1:1234

orbi-monitor: orbi-monitor.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o /tmp/orbi-monitor

deploy: orbi-monitor
	file-transfer send /tmp/orbi-monitor $(ORBI)

upload: orbi-monitor
	gsutil cp /tmp/orbi-monitor gs://alexflint-misc/orbi-monitor
	du -b /tmp/orbi-monitor