package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"

	pb "communication/api"
	"context"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"time"
)

const (
	grpcPort = 50001
	crtFile  = "../certifs/server.pem"
	keyFile  = "../certifs/server-key.pem"
	caFile   = "../certifs/ca.pem"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

type Order struct {
	ID          string
	Items       []string
	Description string
	Price       float32
	Destination string
}

var orderMap = make(map[string]pb.Order)

// oauth2JWT flow where the client which got the user token from the auth service
// send it within any req, which the resource srv should validate
// (in real world: query a single verif endpoint)
func ensureValidJWT(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}

	authHeaders, ok := md["authorization"]
	if !ok || len(authHeaders) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "No authorization header found")
	}

	token := authHeaders[0]

	if ok = validateJWT(token); !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid or expired JWT token")
	}

	return handler(ctx, req)
}

func validateJWT(tokenString string) bool {
	// means that's the entity which verify the token must know the password(with which the token has been created)
	secret := []byte("your-secret-key")

	// remove Bearer
	tokenString = removeBearerPrefix(tokenString)

	// Verify the token
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method and return the secret key
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil // The jwt.Parse function takes a callback function that is responsible for providing the key or secret
	})

	if err != nil {
		fmt.Println("Error verifying token:", err)
		return false
	}

	// Check if the token is valid
	if parsedToken.Valid {
		fmt.Println("Token is valid")
		// Access the claims
		claims := parsedToken.Claims.(jwt.MapClaims)
		fmt.Println("Subject:", claims["sub"])
		fmt.Println("Name:", claims["name"])
		fmt.Println("Email:", claims["email"])
		fmt.Println("Role:", claims["role"])
		fmt.Println("Expiration Time:", time.Unix(int64(claims["exp"].(float64)), 0))
		return true
	} else {
		fmt.Println("Token is invalid")
		return false
	}
}

func removeBearerPrefix(token string) string {
	// Trim any leading or trailing spaces
	token = strings.TrimSpace(token)

	// Check if the token starts with "Bearer "
	if strings.HasPrefix(token, "Bearer ") {
		// Remove the "Bearer " prefix
		token = token[len("Bearer "):]
	}

	return token
}

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
	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}

	// for mTLS ====
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatalf("could not read ca certificate: %s", err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("failed to append ca certificate")
	}

	opt := []grpc.ServerOption{
		// grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		grpc.UnaryInterceptor(ensureValidJWT), // valid the fictif token
		// which actually should be validate with a req to the authentication server
		grpc.Creds(
			credentials.NewTLS(&tls.Config{
				ClientAuth:   tls.RequireAndVerifyClientCert,
				Certificates: []tls.Certificate{cert},
				ClientCAs:    certPool,
			},
			)),
	}
	// ======

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", grpcPort))
	if err != nil {
		log.Fatal("err creating the listener: ", err)
	}

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

func NewGrpcServer(opts ...grpc.ServerOption) (*grpc.Server, error) {
	gsrv := grpc.NewServer(opts...)

	srv := &Server{}

	pb.RegisterOrderManagementServer(gsrv, srv)

	return gsrv, nil
}

func (s *Server) GetOrder(ctx context.Context, orderID *wrapperspb.StringValue) (*pb.Order, error) {
	// srv implementation
	ord := orderMap[orderID.Value]
	return &ord, nil
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
func (s *Server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {

	ordersStr := "Updated Order IDs : "

	ordersLGT := len(orderMap)

	// pbOrders := pb.Order{}

	for {

		order, err := stream.Recv()
		if err == io.EOF {
			// finish reading the order stream
			return stream.SendAndClose(
				&wrapperspb.StringValue{Value: "Orders processed" + ordersStr})
		}

		log.Println("see datas: ", order)

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
