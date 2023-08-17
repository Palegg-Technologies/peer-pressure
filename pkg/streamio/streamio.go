package streamio

import (
	"bufio"
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

func FileToStream(rw *bufio.ReadWriter, file *os.File, eventCh chan peer.Event, cmdCh chan peer.Command) {
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
		handleError(eventCh, err)
		return
	}
	err = rw.Flush()
	if err != nil {
		handleError(eventCh, err)
		return
	}

	cr := &pb.ChunkRequest{}
	pb.Read(rw.Reader, cr)

	partNum := cr.GetIndex()
	newOff, err := file.Seek(int64(partNum/chunkSize), 0)
	log.Debugln("New offset:", newOff)
	if err != nil {
		handleError(eventCh, err)
		return
	}

STREAM_LOOP:
	for {
		_, err := file.Read(data)
		if err == io.EOF {
			break
		} else if err != nil {
			handleError(eventCh, err)
			return
		}

		partNum++
		chunk := &pb.Chunk{
			Index: partNum,
			Data:  data,
		}
		str := pb.Marshal(chunk)
		_, err = rw.Write(str)
		if err != nil {
			handleError(eventCh, err)
			return
		}

		err = rw.Flush()
		if err != nil {
			handleError(eventCh, err)
			return
		}
		pushEvent(eventCh, 1, float64(partNum)*chunkSize/float64(fileInfo.Size()))
		select {
		case cmd := <-cmdCh:
			if cmd == peer.Pause {
				cmd = <-cmdCh
			}
			if cmd == peer.Stop {
				break STREAM_LOOP
			}
		default:
		}
	}
	pushEvent(eventCh, 1, float64(-1))
}

func StreamToFile(rw *bufio.ReadWriter, file *os.File, eventCh chan peer.Event, cmdCh chan peer.Command) {
	indexPath := file.Name() + ".ppindex"
	index := pb.Index{}
	IndexFile, err := os.ReadFile(indexPath)
	if err != nil {
		handleError(eventCh, err)
		return
	}
	defer file.Close()
	proto.Unmarshal(IndexFile, &index)
	if err != nil {
		handleError(eventCh, err)
		return
	}

	writer := bufio.NewWriter(file)
STREAM_LOOP:
	for {
		chunk := &pb.Chunk{}
		err = pb.Read(rw.Reader, chunk)
		if err == io.EOF {
			log.Printf("%s done writing", file.Name())
			break
		} else if err != nil {
			handleError(eventCh, err)
			return
		}
		_, err = writer.Write(chunk.Data)
		if err != nil {
			handleError(eventCh, err)
			return
		}
		index.Progress += 1
		index.Save()

		pushEvent(eventCh, 1, index.Progress)

		select {
		case cmd := <-cmdCh:
			if cmd == peer.Pause {
				cmd = <-cmdCh
			}
			if cmd == peer.Stop {
				break STREAM_LOOP
			}
		default:
		}
	}
	pushEvent(eventCh, 1, int32(-1))

	log.Debugln(writer.Flush())
}

func pushEvent[T int32 | float64 | string](ch chan peer.Event, msgType peer.SignalType, data T) {
	ch <- peer.Event{
		Type: msgType,
		Data: data,
	}
}

func handleError(ch chan peer.Event, err error) {
	pushEvent(ch, peer.Error, err.Error())
}
