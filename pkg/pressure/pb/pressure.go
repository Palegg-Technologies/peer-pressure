package pb

import (
	"encoding/binary"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type pressure interface {
	*Chunk | *ChunkRequest | *Index
	ProtoReflect() protoreflect.Message
}

func (x *Index) Save() {
	indexFile, err := os.OpenFile("nodes/"+x.Filename+".ppindex", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicln("Error creating index file:", err)
	}

	data, err := proto.Marshal(x)
	if err != nil {
		log.Panicln("Error marshaling Index message:", err)
	}

	_, err = indexFile.Write(data)
	if err != nil {
		log.Panicln("Error writing index file:", err)
	}
}

func Read[T pressure](r io.Reader, x T) {
	messageSize, err := readMessageLen(r)
	if err != nil {
		log.Panicf("Error marshaling proto %s message: %v\n", x.ProtoReflect().Type(), err)
	}
	str := make([]byte, messageSize)
	_, err = io.ReadFull(r, str)
	if err != nil {
		log.Panicf("Error reading %s from buffer: %v\n", x.ProtoReflect().Type(), err)
	}

	err = proto.Unmarshal(str, x)
	if err != nil {
		log.Panicf("Error unmarshaling proto %s: %v\n", x.ProtoReflect().Type(), err)
	}
}

func Marshal[T pressure](x T) []byte {
	data, err := proto.Marshal(x)
	if err != nil {
		log.Panicf("Error marshaling proto %s message: %v\n", x.ProtoReflect().Type(), err)
	}

	return append(bigEndianMessageSize(len(data)), data...)
}

func readMessageLen(r io.Reader) (uint32, error) {
	lenBytes := make([]byte, 4)
	n, err := r.Read(lenBytes)
	if err != nil {
		return 0, err
	} else if n == 0 {
		return 0, io.EOF
	}
	return binary.BigEndian.Uint32(lenBytes), nil
}

func bigEndianMessageSize(n int) []byte {
	messageSize := uint32(n)
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)
	return messageSizeBytes
}
