protogen:
	./protoc/bin/protoc -I=. --go_out=. pkg/pressure/pb/pressure.proto

build-all:
	env GOOS=linux GOARCH=arm64 go build -o peer-pressure_linux_arm64 .
	env GOOS=linux GOARCH=amd64 go build -o peer-pressure_linux_amd64 .
	env GOOS=linux GOARCH=386 go build -o peer-pressure_linux_386 .
	env GOOS=darwin GOARCH=arm64 go build -o peer-pressure_darwin_arm64 .
	env GOOS=windows GOARCH=amd64 go build -o peer-pressure_windows_amd64 .
	env GOOS=windows GOARCH=386 go build -o peer-pressure_windows_386 .
	
build-pair:
	go build .
	go build -o ./testdir/peer-pressure .

build-darwin_arm64-pair:
	env GOOS=darwin GOARCH=arm64 go build -o peer-pressure_darwin_arm64 .
	env GOOS=darwin GOARCH=arm64 go build -o ./testdir/peer-pressure_darwin_arm64 .

clearlog-pair:
	> log
	> testdir/log

test-pair: clearlog-pair build-pair
	./peer-pressure