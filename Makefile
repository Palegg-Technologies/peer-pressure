protogen:
	./protoc/bin/protoc -I=. --go_out=. pkg/pressure/pb/pressure.proto

build-all:
	env GOOS=linux GOARCH=arm64 go build -o peer-pressure_linux_arm64 .
	env GOOS=linux GOARCH=amd64 go build -o peer-pressure_linux_amd64 .
	env GOOS=darwin GOARCH=arm64 go build -o peer-pressure_darwin_arm64 .
	
build-pair:
	go build .
	go build -o ./testdir/peer-pressure .

clearlog-pair:
	> log
	> testdir/log