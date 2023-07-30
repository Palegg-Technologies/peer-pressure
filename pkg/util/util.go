package util

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	log "github.com/sirupsen/logrus"

	"github.com/Azanul/peer-pressure/pkg/pressure/pb"
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

func FileToStream(w *bufio.Writer, file *os.File) {
	data := make([]byte, chunkSize)
	filename := filepath.Base(file.Name())
	fileInfo, _ := file.Stat()
	var partNum int32 = 0

	index := &pb.Index{
		NChunks:  int32(math.Ceil(float64(fileInfo.Size()) / chunkSize)),
		Filename: filename,
		Progress: 0,
	}
	_, err := w.Write(index.Marshal())
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
		_, err = w.Write(chunk.Marshal())
		if err != nil {
			log.Println("Error writing to buffer")
			panic(err)
		}

		err = w.Flush()
		if err != nil {
			log.Println("Error flushing buffer")
			panic(err)
		}
	}
}

func StreamToFile(r *bufio.Reader, file *os.File) (filename string) {
	index := pb.Index{}
	index.Read(r)
	index.Save()

	writer := bufio.NewWriter(file)
	for {
		chunk := pb.Chunk{}
		err := chunk.Read(r)
		if err == io.EOF {
			log.Printf("%s done writing", file.Name())
			break
		} else if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

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
	err := writer.Flush()
	if err != nil {
		log.Panic(err)
	}
	return
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
