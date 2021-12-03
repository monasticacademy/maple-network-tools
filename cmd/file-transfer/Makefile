file-transfer: file-transfer.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o /tmp/file-transfer

upload: file-transfer
	gsutil cp /tmp/file-transfer gs://alexflint-misc/file-transfer
