package main

import (
	"fmt"
	"log"
	"net"
	pb "testgrpc/api/v1/server"

	"context"
	"google.golang.org/grpc"
)

const (
	grpcPort = "50200"
)

func main() {

	grpcListen()

}

func grpcListen() {
	fmt.Println("hit grpcListen")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatal("Err listening to grpc: ", err)
	}

	opts := []grpc.ServerOption{}
	// if secure {
	// 	serverCreds := credentials.NewTLS(setTLSCredentials)
	// 	opts = append(opts, grpc.Creds(serverCreds))
	// }

	serv, err := NewGrpcServer(opts...)
	if err != nil {
		log.Fatal("Err setting the server: ", err)
	}

	log.Println("grpc server is listening")
	err = serv.Serve(lis)
	if err != nil {
		log.Fatal("Err server listen: ", err)
	} else {
		log.Println("grpc server listening")
	}

}

type Server struct {
	pb.UnimplementedServerServer
}

func NewGrpcServer(opt ...grpc.ServerOption) (*grpc.Server, error) {

	// gsrv := grpc.NewServer(opt)
	gsrv := grpc.NewServer() // no tls

	svc := &Server{}

	pb.RegisterServerServer(gsrv, svc)

	return gsrv, nil
}

func (s *Server) Get(ctx context.Context, r *pb.GetRequest) (*pb.GetResponse, error) {
	log.Println("the request is: ", r.Request)

	return &pb.GetResponse{Response: "this is the response"}, nil
}
