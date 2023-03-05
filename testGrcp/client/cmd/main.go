package main

import (
	pb "testgrcp/api/v1/client"
	// "fmt"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcPort = "50001"
)

type data struct {
	Response string `json:"response"`
	Request  string `json:"request"`
}

func main() {

	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, ":50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Error when dial: ", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatalln(err)
		}

		// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		// defer cancel()

		dt := data{}
		err = json.Unmarshal(body, &dt)
		if err != nil {
			log.Fatal("err unmarshal: ", err)
		}

		log.Println("see the body: ", dt)

		// clientCreds := credentials.NewTLS(clientTLSConfig)
		// opts := grpc.DialOption{
		// 	grpc.WithTransportCredentials(clientCreds),
		// 	grpc.WithBlock(),
		// 	grpc.WithTimeout(time.Second),
		// }

		client := pb.NewServerClient(conn)

		res, err := client.Get(ctx, &pb.GetRequest{Request: dt.Request})
		if err != nil {
			log.Fatal("Error when send request: ", err)
		}

		log.Println("See the response: ", res.Response)
	})

	// Run the web server.
	log.Println("start producer-api on port 3000 !!")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
