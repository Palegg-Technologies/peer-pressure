package streamio

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/Azanul/peer-pressure/pkg/peer"
	"github.com/Azanul/peer-pressure/pkg/pressure/pb"
	"google.golang.org/protobuf/proto"
)

const chunkSize = 4096

func FileToStream(rw *bufio.ReadWriter, file *os.File, progressCh chan peer.Event) {
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
		progressCh <- peer.Event{
			1,
			float64(partNum) * chunkSize / float64(fileInfo.Size()),
		}
	}
	progressCh <- peer.Event{
		1,
		nil,
	}
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
