protogen:
	./protoc/bin/protoc -I=. --go_out=. pkg/pressure/pb/pressure.proto

build-pair:
	go build .
	go build -o ./testdir/peer-pressure .

clearlog-pair:
	> log
	> testdir/log