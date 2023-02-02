build-multiarch:
	GOOS=windows GOARCH=386 go build -o harvester-ec2-tool-windows-x32.exe .
	GOOS=windows GOARCH=amd64 go build -o harvester-ec2-tool-windows-x64.exe .
	GOOS=linux GOARCH=386 go build -o harvester-ec2-tool-linux-i386 .
	GOOS=linux GOARCH=amd64 go build -o harvester-ec2-tool-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o harvester-ec2-tool-linux-arm64 .
	GOOS=darwin GOARCH=arm64 go build -o harvester-ec2-tool-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o harvester-ec2-tool-darwin-amd64 .