package main

import (
	"VK/config"
	"VK/pkg/subpub"
	pb "VK/proto"
	"VK/server"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Config wasn't loaded: %v", err)
	}
	logLevel, err := logrus.ParseLevel(cfg.Log_level)
	if err != nil {
		log.Fatalf("Config level not valided: %v", err)
	}
	log.SetLevel(logLevel)

	listener, err := net.Listen(cfg.Grpc_server.Network, cfg.Grpc_server.Port)
	if err != nil {
		log.Fatalf("Listner was not announced: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPubSubServer(grpcServer, &server.SubscriptionServiceServer{
		Bus: subpub.NewSubPub(),
		Log: log,
	})

	log.Infof("gRPC server was launched on port: %s", cfg.Grpc_server.Port)
	servErr := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			servErr <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-shutdown:
		log.Infof("Received signal: %v", sig)
	case err := <-servErr:
		log.Fatalf("Server launching error: %v", err)
	}

	grpcServer.GracefulStop()

	log.Info("Graceful shutdown!")
}
