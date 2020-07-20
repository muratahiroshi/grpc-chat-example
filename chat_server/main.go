package main

import (
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"io"
	pb "jp.chat.example/chat"
	"log"
	"net"
	"sync"
)

const (
	port = ":50051"
)

type Room struct {
	streamMap map[string]*pb.Chat_JoinRoomServer
	mux       sync.Mutex
}

func (r *Room) AddStream(id string, stream *pb.Chat_JoinRoomServer) {
	r.mux.Lock()
	r.streamMap[id] = stream
	r.mux.Unlock()
}
func (r *Room) RemoveStream(id string) {
	r.mux.Lock()
	delete(r.streamMap, id)
	r.mux.Unlock()
}
func (r *Room) SendToAllStream(text string) {
	r.mux.Lock()
	for id, stream := range r.streamMap {
		e2 := (*stream).Send(&pb.ChatResponse{Text: text})
		if e2 != nil {
			log.Printf("Failed to send : %v %v\n", id, e2)
		}
	}
	r.mux.Unlock()
}

var room Room

type Server struct {
	pb.UnimplementedChatServer
}

func (s *Server) JoinRoom(stream pb.Chat_JoinRoomServer) error {
	ctx := stream.Context()

	u, err := uuid.NewRandom()
	if err != nil {
		fmt.Println(err)
		ctx.Deadline()
		return err
	}

	id := u.String()
	room.AddStream(id, &stream)
	room.SendToAllStream(fmt.Sprintf("[%sさんが入室しました 現在%d人]", id, len(room.streamMap)))

	for {
		select {
		case <-ctx.Done():
			log.Println("done")
			room.RemoveStream(id)
			room.SendToAllStream(fmt.Sprintf("[%sさんが退室しました 現在%d人]", id, len(room.streamMap)))
			return nil

		default:
			req, e1 := stream.Recv()
			if e1 == io.EOF {
				continue
			}
			if e1 != nil {
				log.Printf("Failed to receive a req : %v\n", e1)
				continue
			}
			switch req.Method {
			case pb.ChatRequest_SEND_MESSAGE:
				room.SendToAllStream(fmt.Sprintf("[%sさんがコメントしました] %s", id, req.Text))
			case pb.ChatRequest_SEND_STAMP:
				room.SendToAllStream(fmt.Sprintf("[%sさんがスタンプを送信しました] スタンプ番号%d", id, req.Number))
			}
		}
	}
}

func main() {
	room.streamMap = make(map[string]*pb.Chat_JoinRoomServer)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterChatServer(s, &Server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
