package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	// "golang.org/x/oauth2"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	pb "communication/api"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	// "google.golang.org/grpc/credentials/oauth"
	// "google.golang.org/grpc/credentials/insecure"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	grpcPort = "50001"
	crtFile  = "../certifs/client.pem"
	keyFile  = "../certifs/client-key.pem"
	caFile   = "../certifs/ca.pem"
	hostname = "127.0.0.1"
)

type Order struct {
	ID          string
	Items       []string
	Description string
	Price       float32
	Destination string
}

var orderMap = make(map[string]pb.Order)

// JWT ------------------
type userJWT struct {
	token string
}

func NewUserJWT() *userJWT {
	token := createJWT()
	return &userJWT{token}
}

func (u *userJWT) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + u.token,
	}, nil
}

func (u *userJWT) RequireTransportSecurity() bool {
	return true
}

func createJWT() string {
	// Create a new token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims (payload), add as many field as needed
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = "1234567890"             // Subject
	claims["name"] = "John Doe"              // Name
	claims["email"] = "john.doe@example.com" // Email claim
	claims["role"] = "user"                  // Role claim
	claims["exp"] = time.Now().Unix() + 60   // Expiration time (1 minute)

	// Sign the token with a secret key
	secret := []byte("your-secret-key")
	tokenString, err := token.SignedString(secret)
	if err != nil {
		fmt.Println("Error creating token:", err)
		return ""
	}

	fmt.Println("JWT:", tokenString)
	return tokenString
}

// ------------------

func main() {
	// in real flow we would get this token from the authentication server
	uJWT := NewUserJWT()

	flg := os.Args[1]

	// for mTLS ====
	certificate, err := tls.LoadX509KeyPair(crtFile, keyFile)
	if err != nil {
		log.Fatalf("could not load client key pair: %s", err)
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatalf("could not read ca certificate: %s", err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("failed to append ca certs")
	}

	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(uJWT), // transport the JWT to the resource srv which will verify it and if valid, then it's ok.
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			ServerName:   hostname, // NOTE: this is required!
			Certificates: []tls.Certificate{certificate},
			RootCAs:      certPool,
		})),
	}
	// =========

	ctx := context.Background()

	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("127.0.0.1:%v", grpcPort),
		opts...,
	// grpc.WithTransportCredentials(insecure.NewCredentials()),
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

// // As we do not have an Authorization server here, create a hardCoded token
// func fetchToken() *oauth2.Token {
// 	return &oauth2.Token{
// 		AccessToken: "some-secret-token",
// 	}
// }

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
