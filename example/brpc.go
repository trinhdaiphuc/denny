package main

import (
	"context"
	"fmt"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/zipkin"
	"github.com/whatvn/denny"
	pb "github.com/whatvn/denny/example/protobuf"
	"github.com/whatvn/denny/middleware/grpc"
	"github.com/whatvn/denny/middleware/http"

	"github.com/whatvn/denny/naming/redis"
)

// grpc
type Hello struct{}

// groupPath + "/hello/" + "say-hello"
//
func (s *Hello) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	var (
		logger = denny.GetLogger(ctx)
	)
	response := &pb.HelloResponse{
		Reply: "hi",
	}

	logger.WithField("response", response)
	return response, nil
}

// http get request
// when define grpc method with input is empty.Empty object, denny will consider request as get request
// router will be:
// groupPath + "/hello/" + "say-hello-anonymous"
// rule is rootRoute + "/" kebabCase(serviceName) + "/" kebabCase(methodName)

func (s *Hello) SayHelloAnonymous(ctx context.Context, in *empty.Empty) (*pb.HelloResponseAnonymous, error) {

	var (
		logger = denny.GetLogger(ctx)
	)

	span, ctx := opentracing.StartSpanFromContext(ctx, "sayHello")
	defer span.Finish()
	response := &pb.HelloResponseAnonymous{
		Reply:  "hoho",
		Status: pb.Status_STATUS_SUCCESS,
	}

	logger.WithField("response", response)

	return response, nil
}

type TestController struct {
	denny.Controller
}

func (t *TestController) Handle(ctx *denny.Context) {
	ctx.JSON(200, &pb.HelloResponse{
		Reply: "ha",
	})
}

func newReporterUDP(jaegerAddr string, port int, packetLength int) jaeger.Transport {
	hostString := fmt.Sprintf("%s:%d", jaegerAddr, port)
	transport, err := jaeger.NewUDPTransport(hostString, packetLength)
	if err != nil {
		panic(err)
	}
	return transport
}
func initTracerUDP(jaegerAddr string, port int, packetLength int, serviceName string) (opentracing.Tracer, io.Closer) {
	var (
		propagator = zipkin.NewZipkinB3HTTPHeaderPropagator()
	)

	return jaeger.NewTracer(
		serviceName,
		jaeger.NewConstSampler(true),
		jaeger.NewRemoteReporter(newReporterUDP(jaegerAddr, port, packetLength)),
		jaeger.TracerOptions.Injector(opentracing.HTTPHeaders, propagator),
		jaeger.TracerOptions.Extractor(opentracing.HTTPHeaders, propagator),
		jaeger.TracerOptions.ZipkinSharedRPCSpan(true),
	)
}

func main() {

	// open tracing
	tracer, _ := initTracerUDP(
		"127.0.0.1",
		6831,
		65000,
		"brpc.server.demo",
	)
	opentracing.SetGlobalTracer(tracer)

	server := denny.NewServer(true)
	server.Use(http.Logger())
	group := server.NewGroup("/hi")
	group.Controller("/hi", denny.HttpPost, new(TestController))

	// setup grpc server

	grpcServer := denny.NewGrpcServer(grpc.ValidatorInterceptor)
	pb.RegisterHelloServiceServer(grpcServer, new(Hello))
	server.WithGrpcServer(grpcServer)
	//

	//// then http
	authorized := server.NewGroup("/")
	// http://127.0.0.1:8080/hello/sayhello  (POST)
	// http://127.0.0.1:8080/hello/sayhelloanonymous  (GET)
	authorized.BrpcController(&Hello{})

	// naming registry
	registry := redis.New("127.0.0.1:6379", "", "demo.brpc.svc")
	server.WithRegistry(registry)

	// start server in dual mode
	server.GraceFulStart(":8080")
}
