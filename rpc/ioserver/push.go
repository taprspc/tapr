// Copyright 2018 Klaus Birkelund Abildgaard Jensen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ioserver

import (
	"io"
	"os"
	"time"

	pb "github.com/golang/protobuf/proto"

	"tapr.space"
	"tapr.space/errors"
	"tapr.space/log"
	"tapr.space/proto"
	"tapr.space/rpc"
)

func (s *server) PushPrepare(reqBytes []byte) (pb.Message, error) {
	var req proto.PushPrepareRequest

	if err := pb.Unmarshal(reqBytes, &req); err != nil {
		return nil, err
	}

	tx := rpc.GenerateTx()

	log.Debug.Printf("rpc/ioserver.PushPrepare (tx: %s): %v", tx, req.Name)

	flags := os.O_CREATE | os.O_WRONLY
	if req.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := s.st.OpenFile(tapr.PathName(req.Name), flags)
	if err != nil {
		return nil, err
	}

	s.mu.fds[tx] = f

	return &proto.PushPrepareResponse{
		Tx: tx[:],
	}, nil
}

func (s *server) Push(body io.Reader, done <-chan struct{}) (pb.Message, error) {
	// read the transaction identifier
	var tx rpc.Tx
	if _, err := rpc.ReadFull(body, tx[:], done); err != nil {
		return nil, err
	}

	log.Debug.Printf("rpc/ioserver.Push (tx: %s): starting", tx)

	stream := make(rpc.ChunkStream)

	go rpc.ReadStream(body, stream, done)

	for {
		select {
		case cnk := <-stream:
			f := s.mu.fds[tx]
			_, err := f.Write(cnk.Data)
			if err != nil {
				log.Debug.Print(err)
				return &proto.PushResponse{Error: errors.MarshalError(err)}, nil
			}

			log.Debug.Printf("rpc/ioserver.Push: received %d bytes", len(cnk.Data))

		case <-done:
			log.Debug.Printf("rpc/ioserver.Push (tx: %s): done; closing file", tx)

			f := s.mu.fds[tx]
			delete(s.mu.fds, tx)
			f.Close()

			return &proto.PushResponse{}, nil
		}
	}
}

func (s *server) PushLog(reqBytes []byte, done <-chan struct{}) (<-chan pb.Message, error) {
	var req proto.PushLogRequest

	if err := pb.Unmarshal(reqBytes, &req); err != nil {
		return nil, err
	}

	tx := rpc.MakeTx(req.Tx)

	log.Debug.Printf("rpc/ioserver.PushLog: (tx: %s)", tx)

	out := make(chan pb.Message)
	go func() {
		defer close(out)

		var i int64

		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-ticker.C:
				out <- &proto.PushLogEntry{Seq: i}
				i++

			case <-done:
				log.Debug.Printf("rpc/ioserver.PushLog: done closed; writer terminating")
				return
			}
		}
	}()

	return out, nil
}
