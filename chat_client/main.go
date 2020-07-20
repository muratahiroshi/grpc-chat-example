package main

import (
	"bufio"
	"context"
	"flag"
	"google.golang.org/grpc"
	"io"
	pb "jp.chat.example/chat"
	"log"
	"os"
)

const (
	address = "localhost:50051"
)

var stdin = bufio.NewScanner(os.Stdin)

func main() {
	flag.Parse()
	addr := address
	if flag.NArg() > 0 {
		addr = flag.Arg(0)
	}
	log.Println("address " + addr)

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewChatClient(conn)

	//timeoutSecond := 60 * time.Second
	//heartbeatSecond := timeoutSecond / 2

	ctx := context.Background()
	//ctx, cancel := context.WithTimeout(context.Background(), timeoutSecond)
	//defer cancel()

	stream, err := c.JoinRoom(ctx)
	if err != nil {
		log.Fatalf("could not JoinRoom: %v", err)
	}

	waitc := make(chan struct{})

	// 受信ループ
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive : %v", err)
			}
			log.Println("Recv " + res.Text)
		}
	}()

	// 標準入力、送信ループ
	for {
		var s string
		if stdin.Scan() {
			s = stdin.Text()
		} else {
			continue
		}

		switch s {
		case "stamp1":
			log.Println("SendStamp1")
			if err := stream.Send(&pb.ChatRequest{Method: pb.ChatRequest_SEND_STAMP, Number: 1}); err != nil {
				log.Fatalf("Failed to send: %v", err)
			}
		case "stamp2":
			log.Println("SendStamp2")
			if err := stream.Send(&pb.ChatRequest{Method: pb.ChatRequest_SEND_STAMP, Number: 2}); err != nil {
				log.Fatalf("Failed to send: %v", err)
			}
		default:
			log.Println("Send " + s)
			if err := stream.Send(&pb.ChatRequest{Method: pb.ChatRequest_SEND_MESSAGE, Text: s}); err != nil {
				log.Fatalf("Failed to send: %v", err)
			}
		}

		if s == "exit" {
			close(waitc)
			break
		}
	}
	stream.CloseSend()
	<-waitc
}
