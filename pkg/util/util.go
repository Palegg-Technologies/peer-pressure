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
	"google.golang.org/protobuf/proto"

	"github.com/Azanul/peer-pressure/pkg/pressure/pb"
)

const chunkSize = 4096

func AppendStringToFile(path string, content string) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	if err != nil {
		return
	}

	return writer.Flush()
}

func FileToStream(rw *bufio.ReadWriter, file *os.File, progressCh chan float64) {
	data := make([]byte, chunkSize)
	filename := filepath.Base(file.Name())
	fileInfo, _ := file.Stat()

	index := &pb.Index{
		NChunks:  int32(math.Ceil(float64(fileInfo.Size()) / chunkSize)),
		Filename: filename,
		Progress: 0,
	}
	str := pb.Marshal(index)
	_, err := rw.Write(str)
	if err != nil {
		log.Println("Error writing to buffer")
		panic(err)
	}
	err = rw.Flush()
	if err != nil {
		log.Println("Error flushing buffer")
		panic(err)
	}

	cr := &pb.ChunkRequest{}
	pb.Read(rw.Reader, cr)

	partNum := cr.GetIndex()
	newOff, err := file.Seek(int64(partNum/chunkSize), 0)
	log.Debugln("New offset:", newOff)
	if err != nil {
		log.Panicln(err)
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
		str := pb.Marshal(chunk)
		_, err = rw.Write(str)
		if err != nil {
			log.Panicln("Error writing to buffer:", err)
		}

		err = rw.Flush()
		if err != nil {
			log.Panicln("Error flushing buffer:", err)
		}
		progressCh <- float64(partNum) * chunkSize / float64(fileInfo.Size())
	}
	progressCh <- -1
}

func StreamToFile(rw *bufio.ReadWriter, file *os.File) (err error) {
	indexPath := file.Name() + ".ppindex"
	index := pb.Index{}
	IndexFile, err := os.ReadFile(indexPath)
	if err != nil {
		return
	}
	defer file.Close()
	proto.Unmarshal(IndexFile, &index)

	writer := bufio.NewWriter(file)
	for {
		chunk := &pb.Chunk{}
		err = pb.Read(rw.Reader, chunk)
		if err == io.EOF {
			log.Printf("%s done writing", file.Name())
			break
		} else if err != nil {
			fmt.Println("Error reading from buffer")
			return err
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
		index.Progress += 1
		index.Save()
	}
	return writer.Flush()
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
