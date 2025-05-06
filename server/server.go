package server

import (
	"VK/pkg/subpub"
	pb "VK/proto"
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SubscriptionServiceServer struct {
	pb.UnimplementedPubSubServer
	Bus subpub.SubPub
	Log *logrus.Logger
}

var empty *emptypb.Empty = &emptypb.Empty{}

func (s *SubscriptionServiceServer) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	sub := req.GetKey()
	msg := req.GetData()
	s.Log.Debugf("Publishing message %s on subject %s", msg, sub)
	if sub == "" {
		s.Log.Error("Invalid argument of subject")
		return empty, status.Errorf(codes.InvalidArgument, "Invalid argument of subject")
	}
	if msg == "" {
		s.Log.Error("Invalid argument of message")
		return empty, status.Errorf(codes.InvalidArgument, "Invalid argument of msg")
	}
	err := s.Bus.Publish(sub, msg)
	if err != nil {
		switch err {
		case subpub.ErrBusClosed:
			s.Log.Warning("Event bus is closed")
			return empty, status.Errorf(codes.Canceled, "Event bus was closed")
		case subpub.ErrSubjectNotFound:
			s.Log.Errorf("Subject %s was not found", sub)
			return empty, status.Errorf(codes.NotFound, "Subject: %s was not found", sub)
		}
	}
	s.Log.Debugf("Publish message %s on subject %s", msg, sub)
	return empty, nil
}

func (s *SubscriptionServiceServer) Subscribe(req *pb.SubscribeRequest, stream grpc.ServerStreamingServer[pb.Event]) error {
	msgChan := make(chan string)
	subject := req.GetKey()
	s.Log.Debugf("Subscribtion on subject: %s", subject)

	if subject == "" {
		s.Log.Error("Invalid argument of subject")
		return status.Errorf(codes.InvalidArgument, "Invalid argument of subject")
	}

	sub, err := s.Bus.Subscribe(subject, func(msg interface{}) {
		strMsg, ok := msg.(string)
		if ok {
			msgChan <- strMsg
		} else {
			s.Log.Error("Invalid message")
		}
	})

	if err != nil {
		if err == subpub.ErrBusClosed {
			s.Log.Warning("Event bus is closed")
			return status.Errorf(codes.Canceled, "Event bus was closed")
		} else {
			s.Log.Errorf("Error of subscription: %v", err)
			return status.Errorf(codes.Internal, "Error of subscription: %v", err)
		}
	}

	s.Log.Debugf("Subscribe on subject: %s", subject)

	for {
		select {
		case msg := <-msgChan:
			if err := stream.Send(&pb.Event{Data: msg}); err != nil {
				s.Log.Errorf("failed to send event: %v", err)
				return status.Errorf(codes.Internal, "failed to send event: %v", err)
			} else {
				s.Log.Debugf("Event %s was sent to stream", msg)
			}
		case <-stream.Context().Done():
			s.Log.Debugf("Stream context of subject %s is done", subject)
			sub.Unsubscribe()
			return nil
		}
	}
}
