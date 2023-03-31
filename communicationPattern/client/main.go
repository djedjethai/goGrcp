package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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

func main() {

	flg := os.Args[1]

	ctx := context.Background()

	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("127.0.0.1:%v", grpcPort),
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

	} else {
		log.Println("Invalid flag.....")
	}
}
