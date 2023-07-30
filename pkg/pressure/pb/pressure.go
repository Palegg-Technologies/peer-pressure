package pb

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"google.golang.org/protobuf/proto"
)

func (x *Chunk) Marshal() []byte {
	data, err := proto.Marshal(x)
	if err != nil {
		log.Println("Error marshaling proto Chunk message")
		panic(err)
	}

	messageSize := uint32(len(data))
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

	return append(messageSizeBytes, data...)
}

func (x *Chunk) Read(r io.Reader) error {
	messageSize, err := readMessageLen(r)
	if err != nil {
		panic(err)
	}

	str := make([]byte, messageSize)
	_, err = io.ReadFull(r, str)
	if err != nil {
		fmt.Println("Error reading from buffer")
		panic(err)
	}

	err = proto.Unmarshal(str, x)
	if err != nil {
		log.Println("Error unmarshaling proto chunk")
		panic(err)
	}
	return nil
}

func (x *ChunkRequest) Marshal() []byte {
	data, err := proto.Marshal(x)
	if err != nil {
		log.Println("Error marshaling proto ChunkRequest message")
		panic(err)
	}

	messageSize := uint32(len(data))
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

	return append(messageSizeBytes, data...)
}

func (x *Index) Marshal() []byte {
	data, err := proto.Marshal(x)
	if err != nil {
		log.Println("Error marshaling proto Index message")
		panic(err)
	}

	messageSize := uint32(len(data))
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

	return append(messageSizeBytes, data...)
}

func (x *Index) Read(r io.Reader) {
	messageSize, err := readMessageLen(r)
	if err != nil {
		panic(err)
	}
	str := make([]byte, messageSize)
	_, err = io.ReadFull(r, str)
	if err != nil {
		log.Println("Error reading from buffer")
		panic(err)
	}

	err = proto.Unmarshal(str, x)
	if err != nil {
		log.Println("Error unmarshaling proto chunk")
		panic(err)
	}
}

func (x *Index) Save() {
	indexFile, err := os.Create(x.Filename + ".ppindex")
	if err != nil {
		log.Println("Error creating index file")
		panic(err)
	}

	data, err := proto.Marshal(x)
	if err != nil {
		log.Println("Error marshaling Index message")
		panic(err)
	}

	_, err = indexFile.Write(data)
	if err != nil {
		log.Println("Error writing index file")
		panic(err)
	}
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
