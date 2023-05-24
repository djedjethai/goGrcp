package main

import (
	pb "communication/api"
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Input struct {
	Mes string
}

type Server2 struct {
	pb.UnimplementedSaySomethingServer
}

func NewGrpcServer2(serv *grpc.Server) {

	srv2 := &Server2{}

	pb.RegisterSaySomethingServer(serv, srv2)
}

func (s *Server2) SayHello(ctx context.Context, say *pb.InputSayHello) (*wrapperspb.StringValue, error) {
	name := Input{
		Mes: say.Mes,
	}

	resp := fmt.Sprintf("Hello %v", name.Mes)

	return &wrapperspb.StringValue{Value: resp}, nil
}
