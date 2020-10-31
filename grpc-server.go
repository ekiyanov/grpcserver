package grpcserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/ekiyanov/logger"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	jaegerlog "github.com/uber/jaeger-client-go/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
)

func main() {
	fmt.Println("vim-go")
}

var gTracer opentracing.Tracer
var tracerMu sync.Mutex

func getOpenTracer(serviceName string) opentracing.Tracer {
	if gTracer == nil {

		tracerMu.Lock()
		cfg := jaegercfg.Configuration{

			ServiceName: serviceName,

			Sampler: &jaegercfg.SamplerConfig{

				Type:  jaeger.SamplerTypeConst,
				Param: 1,
			},
			Reporter: &jaegercfg.ReporterConfig{
				LogSpans: true,
			},
		}

		cfg.FromEnv()

		var jLogger jaegerlog.Logger

		//	if l, ok := logger.(*zap.SugaredLogger); ok {
		//		jLogger = jLogZap.NewLogger(l.Desugar())
		//	} else {
		jLogger = jaegerlog.StdLogger
		//	}
		jMetricsFactory := metrics.NullFactory

		tracer, _, err := cfg.NewTracer(jaegercfg.Logger(jLogger), jaegercfg.Metrics(jMetricsFactory))

		if err != nil {
			log.Println("Unable to create tracer", err)
		} else {
			gTracer = tracer
		}

		tracerMu.Unlock()
	}

	return gTracer

}

func EnableTracingFor(serviceName string) {
	opentracing.SetGlobalTracer(getOpenTracer(serviceName))
}

func DefaultGRPCServer() *grpc.Server {

	logger := Logger()

	opts := []grpc.ServerOption{}

	var sslCert = os.Getenv("SERVER_SSL_CERT")
	var sslKey = os.Getenv("SERVER_SSL_KEY")

	if sslCert != "" {
		creds, err := credentials.NewServerTLSFromFile(sslCert, sslKey)
		if err != nil {
			SLogger().Errorw("Unable to create server credentials", "cert", sslCert, "key", sslKey)
		} else {
			opts = append(opts, grpc.Creds(creds))
		}
	}

	opts = append(opts, grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(),
		grpc_zap.StreamServerInterceptor(logger),
		grpc_recovery.StreamServerInterceptor(),
	)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(logger),
			grpc_recovery.UnaryServerInterceptor(),
		)))

	server := grpc.NewServer(
		opts...,
	)

	return server
}

func ListenGRPC(server *grpc.Server, envvarListen, serviceName string) {
	listenStr := os.Getenv(envvarListen)
	if listenStr == "" {
		SLogger().Debugw("Unable to find listen env var for service. fallback to 0.0.0.0:8080", "serviceName", serviceName, "envVarName", envvarListen)
		listenStr = "0.0.0.0:8080"
	}

	if logger.IsDebug() {
		services := server.GetServiceInfo()
		for k := range services {
			var methods = services[k].Methods
			var mStrings = make([]string, 0, len(methods))
			for _, m := range methods {
				mStrings = append(mStrings, m.Name)
			}
			log.Printf("\n  %v:\n    %v", k, strings.Join(mStrings, "\n    "))
		}
	}

	l, err := net.Listen("tcp", listenStr)
	if err != nil {
		SLogger().Errorw("Failed to listen", "error", err, "serviceName", serviceName, "listen", listenStr)
		return
	}

	SLogger().Debugw("Listening", "serviceName", serviceName, "listen", listenStr)
	err = server.Serve(l)
	if err != nil {
		SLogger().Errorw("Serve error", "error", err, "serviceName", serviceName, "listen", listenStr)
	}
}

// DialGrpc tries to dial grpc service with address
// specified with `envvar` or if it not set with explicitly
// specified address with `defaultAddress`.
// if port is not specified, it would use :8080.
// you have to explicitly specify port :80 if you want to use it
func DialGrpc(ctx context.Context, envvar, defaultAddress string) *grpc.ClientConn {
	address := osGetenv(envvar, defaultAddress)
	//default port :8080
	if strings.Index(address, ":") == -1 {
		address = address + ":8080"
	}

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithInsecure(),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: 300 * time.Millisecond,
		}),
	)
	if err != nil {
		logger.Errorw(ctx, "Failed to Dial GRPC service", "error", err, "address", address)
		panic(err)
	}
	return conn
}
