package util

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"
	"unsafe"

	"github.com/Azanul/peer-pressure/pkg/pressure/pb"
	"google.golang.org/protobuf/proto"
)

const chunkSize = 4096

func AppendStringToFile(path string, content string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	if err != nil {
		log.Panicln(err)
	}

	err = writer.Flush()
	if err != nil {
		log.Fatalln(err)
	}
}

func StreamToFile(rw *bufio.Reader, file *os.File) (filename string) {
	lenBytes := make([]byte, 4)

	n, err := rw.Read(lenBytes)
	if err != nil {
		fmt.Println("Error reading from buffer")
		panic(err)
	}
	messageSize := binary.BigEndian.Uint32(lenBytes)

	str := make([]byte, messageSize)
	_, err = io.ReadFull(rw, str)
	if err != nil {
		fmt.Println("Error reading from buffer")
		panic(err)
	}
	log.Println(n, len(str), str)

	index := pb.Index{}
	err = proto.Unmarshal(str, &index)
	if err != nil {
		log.Println("Error unmarshaling proto chunk")
		panic(err)
	}
	indexFile, err := os.Create(index.GetFilename())
	if err != nil {
		log.Println("Error creating index file")
		panic(err)
	}
	_, err = indexFile.WriteString(index.String())
	if err != nil {
		log.Println("Error writin index file")
		panic(err)
	}

	writer := bufio.NewWriter(file)
	for {
		n, err := rw.Read(lenBytes)
		if err == io.EOF || n == 0 {
			log.Printf("%s done writing", file.Name())
			break
		} else if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}
		messageSize := binary.BigEndian.Uint32(lenBytes)

		str := make([]byte, messageSize)
		_, err = io.ReadFull(rw, str)
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}
		log.Println(n, len(str), str)

		chunk := pb.Chunk{}
		err = proto.Unmarshal(str, &chunk)
		if err != nil {
			log.Println("Error unmarshaling proto chunk")
			panic(err)
		}

		log.Println(chunk.Data)
		log.Println(*chunk.Len)
		log.Println(*chunk.Filename)
		log.Println()
		if chunk.Filename != nil {
			filename = *chunk.Filename
		}

		var data []byte
		if chunk.Len == nil {
			data = chunk.Data
		} else {
			data = chunk.Data[:*chunk.Len]
		}
		_, err = writer.Write(data)
		if err != nil {
			log.Println(err)
		}
	}
	err = writer.Flush()
	if err != nil {
		log.Panic(err)
	}
	return
}

func FileToStream(rw *bufio.Writer, file *os.File) {
	data := make([]byte, chunkSize)
	filename := file.Name()
	fileInfo, _ := file.Stat()

	var partNum int32 = 0
	index := &pb.Index{
		NChunks:  int32(fileInfo.Size() / chunkSize),
		Filename: filename,
		Progress: 0,
	}
	sendData, err := proto.Marshal(index)
	if err != nil {
		log.Println("Error marshaling proto chunk")
		panic(err)
	}

	messageSize := uint32(len(sendData))
	messageSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

	_, err = rw.Write(messageSizeBytes)
	if err != nil {
		log.Println("Error writing size prefix to buffer")
		panic(err)
	}

	_, err = rw.Write(sendData)
	if err != nil {
		log.Println("Error writing to buffer")
		panic(err)
	}

	for {
		n, err := file.Read(data)
		lenData := int32(n)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Error reading from stdin")
			panic(err)
		}

		partNum++
		chunk := &pb.Chunk{
			Index:    partNum,
			Data:     data,
			Filename: &filename,
			Len:      &lenData,
		}
		sendData, err := proto.Marshal(chunk)
		if err != nil {
			log.Println("Error marshaling proto chunk")
			panic(err)
		}

		messageSize := uint32(len(sendData))
		messageSizeBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(messageSizeBytes, messageSize)

		_, err = rw.Write(messageSizeBytes)
		if err != nil {
			log.Println("Error writing size prefix to buffer")
			panic(err)
		}

		_, err = rw.Write(sendData)
		if err != nil {
			log.Println("Error writing to buffer")
			panic(err)
		}
		log.Println(lenData, len(sendData), sendData)
		err = rw.Flush()
		if err != nil {
			log.Println("Error flushing buffer")
			panic(err)
		}
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}
