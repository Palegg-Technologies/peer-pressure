package util

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

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

func StreamToFile(rw *bufio.Reader, file *os.File) {
	writer := bufio.NewWriter(file)
	for {
		str, err := io.ReadAll(rw)
		if err == io.EOF || len(str) == 0 {
			log.Printf("%s done writing", file.Name())
			break
		} else if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}
		chunk := pb.Chunk{}
		err = proto.Unmarshal(str, &chunk)
		if err != nil {
			log.Println("Error unmarshaling proto chunk")
			panic(err)
		}

		log.Println(str)
		log.Println(chunk.Data)
		log.Println()
		_, err = writer.Write(chunk.Data)
		if err != nil {
			log.Println(err)
		}
	}
	err := writer.Flush()
	if err != nil {
		log.Panic(err)
	}
}

func FileToStream(rw *bufio.Writer, file *os.File) {
	data := make([]byte, chunkSize)

	var partNum int32 = 0
	for {
		_, err := file.Read(data)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Error reading from stdin")
			panic(err)
		}

		partNum++
		chunk := &pb.Chunk{
			Index: partNum,
			Data:  data,
		}
		sendData, err := proto.Marshal(chunk)
		if err != nil {
			log.Println("Error marshaling proto chunk")
			panic(err)
		}
		_, err = rw.Write(sendData)
		if err != nil {
			log.Println("Error writing to buffer")
			panic(err)
		}
		log.Println(rw.Available())
		err = rw.Flush()
		if err != nil {
			log.Println("Error flushing buffer")
			panic(err)
		}
	}
}
