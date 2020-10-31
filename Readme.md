### Env vars

 - SERVER_SSL_CERT | to start GRPC server with TLS
 - SERVER_SSL_KEY | required if SERVER_SSL_CERT is specified
 - JAEGER_ENDPOINT | to specify where to send jaeger events [more ar jaeger-client-go](https://github.com/jaegertracing/jaeger-client-go#environment-variables)

### Exposed method
```
  - EnableTracingFor(serviceName string) | Call it before starting server to initialize opent tracer
  - DefaultGRPCServer() | Creates server with ctxTags,openTracing, zapLogger, recoveryFallback
  - ListenGRPC(server *grpc.Server, envvarListen, serviceName string) | start to listen server on address specified by env var envvarListen. envvarListen is not specified, then listens 0.0.0.0:8080. serviceName used to log on zap logger 
  - DialGrpc(ctx context.Context, envvar, defaultAddress string) *grpc.ClientConn
```
