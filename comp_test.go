package grpc_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pb "grpctest/grpctest"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/johnsiilver/golib/ipc/uds"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getClient(socketAddr string) (client *uds.Client) {
	if socketAddr == "" {
		fmt.Println("did not pass --addr")
		os.Exit(1)
	}

	cred, _, err := uds.Current()
	if err != nil {
		panic(err)
	}

	// Connects to the server at socketAddr that must have the file uid/gid of
	// our current user and one of the os.FileMode specified.
	client, err = uds.NewClient(socketAddr, cred.UID.Int(), cred.GID.Int(), []os.FileMode{0770, 1770})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return client
}

func startServer(ctx context.Context, socketAddr string) {
	// fmt.Println(" going to start server")
	cred, _, err := uds.Current()
	if err != nil {
		panic(err)
	}

	// This will set the socket file to have a uid and gid of whatever the
	// current user is. 0770 will be set for the file permissions (though on some
	// systems the sticky bit gets set, resulting in 1770.
	serv, err := uds.NewServer(socketAddr, cred.UID.Int(), cred.GID.Int(), 0770)
	if err != nil {
		panic(err)
	}

	// fmt.Println("Listening on socket: ", socketAddr)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case conn := <-serv.Conn():
				conn1 := conn

				// We spinoff handling of this connection to its own goroutine and
				// go back to listening for another connection.
				go func() {
					defer conn1.Close()
					buff := make([]byte, 1024*1024)
					siz, _ := conn.Read(buff)
					msg := &pb.HeadersAndFirstChunk{}
					proto.Unmarshal(buff[:siz], msg)

					// Write to the stream every 10 seconds until the connection closes.
					if _, err := conn1.Write([]byte(fmt.Sprintf("%s\n size:%d", time.Now().UTC(), siz))); err != nil {
					}

				}()
			}
		}
	}()
}

func benchmarkUDSProto(b *testing.B) {
	b.StopTimer()

	ctxt, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	socketAddr := filepath.Join(os.TempDir(), uuid.New().String())
	startServer(ctxt, socketAddr)
	client := getClient(socketAddr)

	stuff := getRequest()

	data, _ := proto.Marshal(stuff)
	buff := make([]byte, 1024)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		client.Write(data)
		siz, _ := client.Read(buff)
		_ = buff[:siz]
		// fmt.Println(" got output ", fmt.Sprint(string(buff)))
	}
}

func sinkCall(stuff *pb.HeadersAndFirstChunk) (output string) {
	siz := len(stuff.String())
	output = fmt.Sprintf("%s\n size:%d", time.Now().UTC(), siz)
	return
}

func benchmarkFunCall(b *testing.B) {
	b.StopTimer()

	stuff := getRequest()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		result := sinkCall(stuff)
		_ = result
		// fmt.Println(" output ", result)
		// fmt.Println(" got output ", fmt.Sprint(string(buff)))
	}
}

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedHelloServer
}

// Echo implements helloworld.GreeterServer
func (s *server) Echo(ctx context.Context, in *pb.HeadersAndFirstChunk) (*pb.Response, error) {
	return &pb.Response{
		Something: fmt.Sprintf("size :%d", len(in.String())),
		Else:      fmt.Sprintf("time: %s", time.Now().UTC()),
	}, nil
}

func startgRPCServer(ctxt context.Context, connType string, sockAddr string) {
	flag.Parse()
	lis, err := net.Listen(connType, sockAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterHelloServer(s, &server{})

	go func() {
		<-ctxt.Done()
		s.Stop()
	}()

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
}

func getgRPCClient(ctx context.Context, connType string, sockaddr string) (client pb.HelloClient) {
	credentials := insecure.NewCredentials() // No SSL/TLS

	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, connType, sockaddr)
	}

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials),
		grpc.WithBlock(),
		grpc.WithContextDialer(dialer),
	}

	conn, err := grpc.Dial(sockaddr, options...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	return pb.NewHelloClient(conn)
}

func getRequest() *pb.HeadersAndFirstChunk {
	headerPairs := make([]*pb.HeaderPair, 1)
	for i := 0; i < 10; i++ {
		headerPairs = append(headerPairs, &pb.HeaderPair{
			Key:   []byte(strings.Repeat("k", 32)),
			Value: []byte(strings.Repeat("v", 512)),
		})
	}

	return &pb.HeadersAndFirstChunk{
		TransactionID: strings.Repeat("a", 128),
		RemoteAddr:    strings.Repeat("1", 128),
		ConfigID:      strings.Repeat("a", 128),
		Method:        "POST",
		Uri:           strings.Repeat("f", 4096),
		UriParsed:     []byte(strings.Repeat("f", 4096)),
		Protocol:      strings.Repeat("a", 10),
		QueryString:   strings.Repeat("f", 4096),
		ProtoNum:      strings.Repeat("a", 10),
		RequestLine:   strings.Repeat("f", 4096),
		FileName:      strings.Repeat("a", 128),
		Headers:       headerPairs,
	}
}

func benchmarkGRPCUdsProto(b *testing.B) {
	b.StopTimer()
	ctxt, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	socketAddr := filepath.Join(os.TempDir(), uuid.New().String())
	connType := "unix"

	startgRPCServer(ctxt, connType, socketAddr)
	client := getgRPCClient(ctxt, connType, socketAddr)

	stuff := getRequest()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		r, _ := client.Echo(ctx, stuff)
		_ = r
		// fmt.Println("err is ", err)
		// fmt.Println("got output", r)
	}
}

func benchmarkGRPCTcpProto(b *testing.B) {
	b.StopTimer()
	ctxt, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	socketAddr := ":38596"
	connType := "tcp"

	startgRPCServer(ctxt, connType, socketAddr)
	client := getgRPCClient(ctxt, connType, socketAddr)

	stuff := getRequest()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		r, _ := client.Echo(ctx, stuff)
		_ = r
		// fmt.Println("err is ", err)
		// fmt.Println("got output", r)
	}
}

func BenchmarkCompare(b *testing.B) {
	b.Run("Function call, proto  ", func(b *testing.B) {
		benchmarkFunCall(b)
	})

	b.Run("Call through UDS,proto", func(b *testing.B) {
		benchmarkUDSProto(b)
	})

	b.Run("Call through GRPC,tcp,proto", func(b *testing.B) {
		benchmarkGRPCTcpProto(b)
	})

	b.Run("Call through GRPC,uds,proto", func(b *testing.B) {
		benchmarkGRPCUdsProto(b)
	})
}
