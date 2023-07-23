package util

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

const chunkSize = 2048

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

func ReadFromStream(rw *bufio.Reader, saveFilePath string) {
	f, _ := os.Create(saveFilePath)
	defer f.Close()

	str := make([]byte, 2048)
	writer := bufio.NewWriter(f)
	for {
		_, err := rw.Read(str)
		if err == io.EOF {
			log.Printf("%s done writing", saveFilePath)
			break
		} else if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}
		_, err = writer.Write(str)
		if err != nil {
			log.Println(err)
		}
	}
	writer.Flush()
}

func WriteToStream(rw *bufio.Writer, sharedFilePath string) {
	f, _ := os.Open(sharedFilePath)
	defer f.Close()

	sendData := make([]byte, chunkSize)

	for {
		_, err := f.Read(sendData)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Error reading from stdin")
			panic(err)
		}

		log.Println(sendData)
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
