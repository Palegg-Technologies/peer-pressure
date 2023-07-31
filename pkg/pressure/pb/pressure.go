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
		log.Panicln("Error marshaling proto Chunk message:", err)
	}

	messageSize := uint32(len(data))
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

	return append(messageSizeBytes, data...)
}

func (x *Chunk) Read(r io.Reader) error {
	messageSize, err := readMessageLen(r)
	if err != nil {
		return fmt.Errorf("error reading length of Chunk message: %v", err)
	}

	str := make([]byte, messageSize)
	_, err = io.ReadFull(r, str)
	if err != nil {
		return fmt.Errorf("error reading from buffer: %v", err)
	}

	err = proto.Unmarshal(str, x)
	if err != nil {
		return fmt.Errorf("error unmarshaling proto chunk: %v", err)
	}
	return nil
}

func (x *ChunkRequest) Marshal() []byte {
	data, err := proto.Marshal(x)
	if err != nil {
		log.Panicln("Error marshaling proto ChunkRequest message:", err)
	}

	messageSize := uint32(len(data))
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

	return append(messageSizeBytes, data...)
}

func (x *Index) Marshal() []byte {
	data, err := proto.Marshal(x)
	if err != nil {
		log.Panicln("Error marshaling proto Index message:", err)
	}

	messageSize := uint32(len(data))
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

	return append(messageSizeBytes, data...)
}

func (x *Index) Read(r io.Reader) {
	messageSize, err := readMessageLen(r)
	if err != nil {
		log.Panicln("Error marshaling proto Index message:", err)
	}
	str := make([]byte, messageSize)
	_, err = io.ReadFull(r, str)
	if err != nil {
		log.Panicln("Error reading from buffer:", err)
	}

	err = proto.Unmarshal(str, x)
	if err != nil {
		log.Panicln("Error unmarshaling proto chunk:", err)
	}
}

func (x *Index) Save() {
	indexFile, err := os.Create("nodes/" + x.Filename + ".ppindex")
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
