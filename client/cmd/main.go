package main

import (
	"context"
	"log"
	"time"

	pb "client/proto"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}
	defer conn.Close()

	client := pb.NewPubSubClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	stream1, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: "1"})
	if err != nil {
		log.Fatalf("Ошибка при подписке: %v", err)
	}
	time.Sleep(time.Millisecond * 1)

	_, err = client.Publish(ctx, &pb.PublishRequest{Key: "1", Data: "Hellooooo"})
	if err != nil {
		log.Fatalf("Ошибка публикации: %v", err)
	}

	event, _ := stream1.Recv()
	log.Printf("Получено событие: %s", event.GetData())
}
