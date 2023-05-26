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
	"google.golang.org/grpc/resolver"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// const grpcPort = "50001"

type Order struct {
	ID          string
	Items       []string
	Description string
	Price       float32
	Destination string
}

var orderMap = make(map[string]pb.Order)

var (
	exampleServiceName string = "myservice.com"
	addr               string = fmt.Sprintf("example:///%s", exampleServiceName)
)

// register the servers' url matching the service name
// (here I have only one but I could have 5)
var addrsStore = map[string][]string{
	"myservice.com": {
		"127.0.0.1:50001",
	},
}

// all the DNS logic works out of the box.... crazy
type exampleResolverBuilder struct{}

func (*exampleResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {

	r := &exampleResolver{
		target:     target,
		cc:         cc,
		addrsStore: addrsStore,
	}
	r.start()
	return r, nil
}

func (*exampleResolverBuilder) Scheme() string {
	return "example"
}

type exampleResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *exampleResolver) start() {

	addrStrs := r.addrsStore[r.target.Endpoint()]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (*exampleResolver) ResolveNow(o resolver.ResolveNowOptions) {}

func (*exampleResolver) Close() {}

// The init func is automaticaly triggered at program start
func init() {
	resolver.Register(&exampleResolverBuilder{})
}

func main() {

	flg := os.Args[1]

	ctx := context.Background()

	conn, err := grpc.DialContext(
		ctx,
		// fmt.Sprintf("127.0.0.1:%v", grpcPort),
		addr, // example:///myservice.com
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("Err dialing: ", err)
	}

	client := pb.NewOrderManagementClient(conn)

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

		time.Sleep(time.Millisecond * 3000)

		// time.Sleep(time.Millisecond * 1000)

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
