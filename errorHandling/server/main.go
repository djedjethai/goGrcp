package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"
	// "strings"

	pb "communication/api"
	"context"

	// "github.com/golang/protobuf/proto"
	// "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		Price:       23,
		Destination: "Thailand",
	}

	orderMap["1"] = pb.Order{
		Id:          "456",
		Items:       []string{"third", "forth"},
		Description: "decribeeee again",
		Price:       23,
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

	opt := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(10),
	}

	serv, err := NewGrpcServer(orderUnaryServerInterceptor, opt)
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

func NewGrpcServer(interceptor grpc.UnaryServerInterceptor, opt []grpc.ServerOption) (*grpc.Server, error) {
	gsrv := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor),
		grpc.StreamInterceptor(orderServerStreamInterceptor),
		// opt..., // does not work, what the fuck...
	)

	srv := &Server{}

	pb.RegisterOrderManagementServer(gsrv, srv)

	return gsrv, nil
}

func (s *Server) GetOrder(ctx context.Context, orderID *wrapperspb.StringValue) (*pb.Order, error) {
	fmt.Println("In GetOrder...")
	// srv implementation(client req expire at 2s)
	// time.Sleep(3000 * time.Millisecond)

	// implement a logic to react to the clientDeadLine added at the client
	switch err := ctx.Err(); err {
	case context.DeadlineExceeded:
		// Return a deadline exceeded error to the client
		return nil, status.Error(codes.DeadlineExceeded, "operation timed out")
	case context.Canceled:
		// Return a canceled error to the client
		return nil, status.Error(codes.Canceled, "operation canceled")
	default:
		// Continue with the RPC method logic

		// TODO handle error
		ord, ok := orderMap[orderID.Value]
		if !ok {
			errorStatus := status.New(codes.InvalidArgument, "Invalid information received")
			ds, err := errorStatus.WithDetails(
				&errdetails.BadRequest_FieldViolation{
					Field: "ID",
					Description: fmt.Sprintf(
						"Order ID received is not valid %s", orderID.Value),
				},
			)
			if err != nil {
				return nil, errorStatus.Err()
			}

			// will output: rpc error: code = InvalidArgument desc = Invalid information received
			return nil, ds.Err()
		}
		return &ord, nil
	}

}

// Unary Server Interceptor, that will works with GetOrder
func orderUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Preprocessing logic
	// Gets info about the current RPC call by examining the args passed in
	// here the req reach the service
	log.Println("======= [Server Interceptor] ", info.FullMethod)

	// Invoking the handler to complete the normal execution of a unary RPC.
	// here the req go to the handler
	m, err := handler(ctx, req)

	// Post processing logic
	// here the req go back to the client
	log.Printf(" Post Proc Message : %s", m)
	return m, err
}

// return a stream with found datas
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

// receive a stream from the client and update its datas
func (s *Server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) pb.UpdateOrdersResponse {

	ordersLGT := len(orderMap)

	// pbOrders := pb.Order{}

	for {

		order, err := stream.Recv()
		if err == io.EOF {
			// finish reading the order stream
			return &wrapperspb.StringValue{Value: "Orders processed"}
			break
		}
		log.Println("see datas: ", order)

		if order.Price == 0.0 {
			err = errors.New("Missing field(s)")
		}

		// TODO if all field of an order are not complete send an error
		// error which will be handled by the client(cf see the client)
		if err != nil {
			return &wrapperspb.StringValue{Value: ""}, pb.ErrorServer{Key: "Price"}.MissingField()
		}

		orderMap[strconv.Itoa(ordersLGT)] = *order
		ordersLGT++
	}
}

// stream bidirectional
func (s *Server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {
	var batchMarker = 3
	var orderBatchSize = 0

	var combinedShipmentMap = make(map[string]pb.CombinedShipment)
	pbt := pb.CombinedShipment{
		Id:     "1",
		Status: "SendingToThailand",
	}
	combinedShipmentMap["Thai"] = pbt
	pbf := pb.CombinedShipment{
		Id:     "2",
		Status: "SendingToFrance",
	}
	combinedShipmentMap["France"] = pbf

	var ordersFr = []*pb.Order{}
	var ordersTh = []*pb.Order{}
	// var newOrder Order

	for {
		orderID, err := stream.Recv()
		if err == io.EOF {

			for _, comb := range combinedShipmentMap {
				stream.Send(&comb)
			}
			return nil
		}

		// TODO handle the err "canceled",
		// which has been explicitly triggered from the client

		// else if status, ok := status.FromError(err); ok && status.Code() == codes.Canceled {
		// 	fmt.Println("Handle the triggered err 'canceled' from the client")

		// }

		if err != nil {
			return err
		}

		// Logic to organize orders into shipments,
		newOrder := orderMap[orderID.Value]

		if newOrder.Destination == "France" {
			ordersFr = append(ordersFr, &newOrder)
			pbf.OrdersList = ordersFr
			orderBatchSize++
		} else if newOrder.Destination == "Thailand" {
			ordersTh = append(ordersTh, &newOrder)
			pbt.OrdersList = ordersTh
			orderBatchSize++
		} else {
			log.Println("unknow Destination: ", newOrder.Destination)
		}

		// based on the destination.

		combinedShipmentMap["France"] = pbf
		combinedShipmentMap["Thai"] = pbt

		if batchMarker == orderBatchSize {
			for _, comb := range combinedShipmentMap {
				stream.Send(&comb)
			}
			// reset all var
			orderBatchSize = 0
			combinedShipmentMap = make(map[string]pb.CombinedShipment)
			ordersFr = []*pb.Order{}
			ordersTh = []*pb.Order{}
		}
	}
}

// the stream interceptor
type wrappedStream struct {
	grpc.ServerStream
}

func newWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s}
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] "+
		"Receive a message (Type: %T) at %s",
		m, time.Now().Format(time.RFC3339))
	return w.ServerStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] "+
		"Send a message (Type: %T) at %v",
		m, time.Now().Format(time.RFC3339))
	return w.ServerStream.SendMsg(m)
}

func orderServerStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	log.Println("====== [Server Stream Interceptor] ", info.FullMethod)

	err := handler(srv, newWrappedStream(ss))
	if err != nil {
		log.Printf("RPC failed with error %v", err)
	}

	return err
}
