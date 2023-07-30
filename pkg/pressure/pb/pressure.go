package pb

import (
	"encoding/binary"
	"log"

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
