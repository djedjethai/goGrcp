package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	pb "communication/api"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const grpcPort = "50001"

type Order struct {
	ID          string
	Items       []string
	Description string
	Price       float32
	Destination string
}

var orderMap = make(map[string]pb.Order)

func main() {

	flg := os.Args[1]

	ctx := context.Background()

	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("127.0.0.1:%v", grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(orderUnaryClientInterceptor),
		grpc.WithStreamInterceptor(clientStreamInterceptor),
	)
	if err != nil {
		log.Fatalln("Err dialing: ", err)
	}

	client := pb.NewOrderManagementClient(conn)

	// TODO with deadLine
	// clientDeadline := time.Now().Add(time.Duration(2 * time.Second))
	// the context is pass in each client.Req() their the DeadLine will be apply
	// ctx, cancel := context.WithDeadline(context.Background(), clientDeadline)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if strings.Compare(flg, "order") == 0 {

		res, err := client.GetOrder(ctx, &wrapperspb.StringValue{Value: "1"})
		if err != nil {
			log.Fatalln("Err from client: ", err)
		}

		ord := Order{
			ID:          res.Id,
			Items:       res.Items,
			Description: res.Description,
			Price:       res.Price,
			Destination: res.Destination,
		}

		log.Println("the ord: ", ord)

	} else if strings.Compare(flg, "stream") == 0 {
		fmt.Println("let start....")
		res, err := client.SearchOrders(ctx, &wrapperspb.StringValue{Value: "0"})
		if err != nil {
			log.Fatalln("Err from stream client: ", err)
		}

		for {
			orders, err := res.Recv()
			if err == io.EOF {
				break
			}
			log.Println("See the result: ", orders)
		}
		// client send a stream to server
	} else if strings.Compare(flg, "add") == 0 {
		orderMap["0"] = pb.Order{
			Id:          "aaa",
			Items:       []string{"firyyyyy", "secondyyyy"},
			Description: "YUI",
			Price:       233456.4,
			Destination: "Cambodge",
		}

		orderMap["1"] = pb.Order{
			Id:          "hjk",
			Items:       []string{"thYYY", "forthYYY"},
			Description: "from where",
			Price:       9,
			Destination: "Netherland",
		}

		updateStream, err := client.UpdateOrders(ctx)
		if err != nil {
			log.Fatalln("Err from stream client: ", err)
		}

		for _, v := range orderMap {
			log.Println("see the data sent: ", v)

			if err := updateStream.Send(&v); err != nil {
				log.Println("Err client stream sending datas: ", err)
			}
		}

		updateRes, err := updateStream.CloseAndRecv()
		if err != nil {
			log.Fatalln("Err close stream client: ", err)
		}

		log.Printf("Update Orders Res : %s", updateRes)

	} else if strings.Compare(flg, "concat") == 0 {
		streamProcOrder, err := client.ProcessOrders(ctx)
		if err != nil {
			log.Fatalln("Err from stream client: ", err)
		}

		ids := []string{"0", "1", "0", "0", "1", "1", "0", "1"}

		for _, v := range ids {
			if err := streamProcOrder.Send(&wrapperspb.StringValue{Value: v}); err != nil {
				log.Println("Err client stream sending ids: ", err)
			}
		}

		channel := make(chan struct{})
		go asyncBidirectionalRPC(streamProcOrder, channel)

		time.Sleep(time.Millisecond * 1000)

		// TODO here is the client cancellation implementation
		// Canceling the RPC, we call the cancel() on purpose
		// which will indicate to the client and the server that the communication
		// has been canceled
		cancel()
		log.Printf("RPC Status : %s", ctx.Err())

		ids = []string{"0", "1", "0", "0"}
		for _, v := range ids {
			if err := streamProcOrder.Send(&wrapperspb.StringValue{Value: v}); err != nil {
				log.Println("Err client stream sending ids: ", err)
			}
		}

		<-channel

	} else {
		log.Println("Invalid flag.....")
	}
}

func asyncBidirectionalRPC(
	streamProcOrder pb.OrderManagement_ProcessOrdersClient,
	c chan struct{}) {
	for {
		combinedShipment, err := streamProcOrder.Recv()
		if err == io.EOF {
			log.Println("Combined shipment err: ", err)
			break
		}
		// TODO pass the datas throught the channel
		log.Println("received datas: ", combinedShipment)
	}

	<-c
}

// Unary client interceptor
func orderUnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Preprocessor phase
	log.Println("Method(client) : " + method)

	// Invoking the remote method
	err := invoker(ctx, method, req, reply, cc, opts...)

	// Postprocessor phase
	log.Println("postprocessing(client): ", reply)

	return err
}

// stream client interceptor
func clientStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	log.Println("======= [Client Interceptor] ", method)
	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}
	return newWrappedStream(s), nil
}

type wrappedStream struct {
	grpc.ClientStream
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor] "+
		"Receive a message (Type: %T) at %v",
		m, time.Now().Format(time.RFC3339))
	return w.ClientStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor] "+
		"Send a message (Type: %T) at %v",
		m, time.Now().Format(time.RFC3339))
	return w.ClientStream.SendMsg(m)
}

func newWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStream{s}
}
