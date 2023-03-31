package main

import (
	"fmt"
	"log"
	"net"
	// "strings"

	pb "communication/api"
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	grpcPort = 50001
)

type Order struct {
	ID          string
	Items       []string
	Description string
	Price       float32
	Destination string
}

var orderMap = make(map[string]pb.Order)

func main() {

	orderMap["0"] = pb.Order{
		Id:          "123",
		Items:       []string{"first", "second"},
		Description: "decribeeee",
		Price:       23.4,
		Destination: "Thailand",
	}

	orderMap["1"] = pb.Order{
		Id:          "456",
		Items:       []string{"third", "forth"},
		Description: "decribeeee again",
		Price:       82.5,
		Destination: "France",
	}

	fmt.Println("vim-go")
	grpcListen()
}

func grpcListen() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", grpcPort))
	if err != nil {
		log.Fatal("err creating the listener: ", err)
	}

	opt := []grpc.ServerOption{}

	serv, err := NewGrpcServer(opt...)
	if err != nil {
		log.Fatal("err creating the server: ", err)
	}

	log.Println("Server is listening")
	err = serv.Serve(lis)
	if err != nil {
		log.Println("err server listen: ", err)
	}

}

type Server struct {
	pb.UnimplementedOrderManagementServer
}

func NewGrpcServer(opt ...grpc.ServerOption) (*grpc.Server, error) {
	gsrv := grpc.NewServer()

	srv := &Server{}

	pb.RegisterOrderManagementServer(gsrv, srv)

	return gsrv, nil
}

func (s *Server) GetOrder(ctx context.Context, orderID *wrapperspb.StringValue) (*pb.Order, error) {
	// srv implementation
	ord := orderMap[orderID.Value]
	return &ord, nil
}

// p 66
func (s *Server) SearchOrders(orderID *wrapperspb.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {
	for key, order := range orderMap {
		// for _, item := range order.Items {
		// if strings.Contains(item, orderID.Value) {
		err := stream.Send(&order)
		if err != nil {
			return fmt.Errorf("error sending msg to stream: %v", err)
		}
		log.Print("Matching Order Found : " + key)
		// break
		// }
		// }
	}
	return nil
}
