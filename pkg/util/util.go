package util

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

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
	writer := bufio.NewWriter(f)
	for {
		str, err := rw.ReadByte()
		if err == io.EOF {
			log.Printf("%s done writing", saveFilePath)
			break
		} else if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}
		fmt.Println(str)
		err = writer.WriteByte(str)
		if err != nil {
			log.Println(err)
		}
	}
	writer.Flush()
}

func WriteToStream(rw *bufio.Writer, sharedFilePath string) {
	f, _ := os.Open(sharedFilePath)
	stdReader := bufio.NewReader(f)
	defer f.Close()

	for {
		sendData, err := stdReader.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Error reading from stdin")
			panic(err)
		}

		log.Println(sendData)
		err = rw.WriteByte(sendData)
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
