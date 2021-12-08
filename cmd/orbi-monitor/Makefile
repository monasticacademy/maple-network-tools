default: build/orbi-monitor

build/orbi-monitor: orbi-monitor.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o build/orbi-monitor
	du -b build/orbi-monitor

serve:
	python3 -m http.server --directory ./build 8000

# previously I would upload to google storage and download from orbi but
# this is too slow on MAPLE network
upload: build/orbi-monitor
	gsutil cp build/orbi-monitor gs://alexflint-misc/orbi-monitor
	du -b build/orbi-monitor
