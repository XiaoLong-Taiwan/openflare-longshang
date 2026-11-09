package trace

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestOTLPHTTPTraceExportHeadersAndResources(t *testing.T) {
	type received struct {
		headers http.Header
		data    []byte
		path    string
		method  string
	}
	requests := make(chan received, 1)
	responseBody, err := proto.Marshal(&collecttrace.ExportTraceServiceResponse{})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		requests <- received{headers: r.Header.Clone(), data: body, path: r.URL.Path, method: r.Method}
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(responseBody); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "X-API-Key=test-key,X-Client-Name=test%20client")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.namespace=example,deployment.environment.name=test")

	for _, tt := range []struct {
		name        string
		env         map[string]string
		wantPath    string
		wantHeaders map[string]string
	}{
		{
			name:        "base_endpoint",
			wantPath:    "/v1/traces",
			wantHeaders: map[string]string{"X-API-Key": "test-key", "X-Client-Name": "test client"},
		},
		{
			name:        "base_endpoint_path",
			env:         map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": server.URL + "/collector"},
			wantPath:    "/collector/v1/traces",
			wantHeaders: map[string]string{"X-API-Key": "test-key", "X-Client-Name": "test client"},
		},
		{
			name: "trace_specific_overrides",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": server.URL + "/custom/ingest",
				"OTEL_EXPORTER_OTLP_TRACES_HEADERS":  "X-API-Key=trace-key,X-Client-Name=trace%20client",
			},
			wantPath:    "/custom/ingest",
			wantHeaders: map[string]string{"X-API-Key": "trace-key", "X-Client-Name": "trace client"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			provider, err := newTracerProvider(Config{AppName: "openflare-control-plane", SamplingRate: 1, ExportProtocol: "http/protobuf"})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Error(err)
				}
			})
			_, span := provider.Tracer("test").Start(context.Background(), "canary")
			span.End()
			if err := provider.ForceFlush(context.Background()); err != nil {
				t.Fatal(err)
			}
			select {
			case req := <-requests:
				if got := req.headers.Get("Content-Type"); got != "application/x-protobuf" {
					t.Errorf("OTLP Content-Type = %q, want application/x-protobuf", got)
				}
				for key, want := range tt.wantHeaders {
					if got := req.headers.Get(key); got != want {
						t.Errorf("OTLP request header %q = %q, want %q", key, got, want)
					}
				}
				if req.method != "POST" || req.path != tt.wantPath {
					t.Errorf("OTLP request = %s %s, want POST %s", req.method, req.path, tt.wantPath)
				}
				var payload collecttrace.ExportTraceServiceRequest
				if err := proto.Unmarshal(req.data, &payload); err != nil {
					t.Fatal(err)
				}
				if len(payload.ResourceSpans) == 0 {
					t.Fatal("missing resource spans")
				}
				attrs := make(map[string]string)
				for _, a := range payload.ResourceSpans[0].Resource.Attributes {
					attrs[a.Key] = a.Value.GetStringValue()
				}
				for key, want := range map[string]string{"service.name": "openflare-control-plane", "service.namespace": "example", "deployment.environment.name": "test"} {
					if got := attrs[key]; got != want {
						t.Errorf("resource attribute %s = %q, want %q", key, got, want)
					}
				}
			default:
				t.Fatal("OTLP HTTP request was not received")
			}
		})
	}
}

func TestUnknownOTLPProtocolFails(t *testing.T) {
	if _, err := newTraceExporter(context.Background(), "http/json"); err == nil {
		t.Fatal("newTraceExporter(http/json) succeeded, want unsupported protocol error")
	}
}

func TestDefaultOTLPProtocolGRPC(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	requests := make(chan *collecttrace.ExportTraceServiceRequest, 1)
	server := grpc.NewServer()
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "opentelemetry.proto.collector.trace.v1.TraceService",
		HandlerType: (*collecttrace.TraceServiceServer)(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Export",
			Handler: func(_ any, _ context.Context, decode func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
				var request collecttrace.ExportTraceServiceRequest
				if err := decode(&request); err != nil {
					return nil, err
				}
				requests <- &request
				return &collecttrace.ExportTraceServiceResponse{}, nil
			},
		}},
	}, nil)
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Errorf("gRPC collector Serve() = %v, want nil", err)
		}
	}()
	t.Cleanup(server.Stop)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://"+listener.Addr().String())
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_INSECURE", "true")
	for _, protocol := range []string{"", "grpc", " GRPC "} {
		exporter, err := newTraceExporter(context.Background(), protocol)
		if err != nil {
			t.Fatalf("newTraceExporter(%q): %v", protocol, err)
		}
		provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, span := provider.Tracer("grpc-test").Start(ctx, "grpc-canary")
		span.End()
		if err := provider.Shutdown(ctx); err != nil {
			t.Errorf("provider.Shutdown(%q) = %v, want nil", protocol, err)
		}
		cancel()
		select {
		case req := <-requests:
			if len(req.ResourceSpans) != 1 || len(req.ResourceSpans[0].ScopeSpans) != 1 || len(req.ResourceSpans[0].ScopeSpans[0].Spans) != 1 {
				t.Fatalf("gRPC Export(%q) = %v, want one exported span", protocol, req)
			}
			if got := req.ResourceSpans[0].ScopeSpans[0].Spans[0].Name; got != "grpc-canary" {
				t.Errorf("gRPC Export(%q).Name = %q, want grpc-canary", protocol, got)
			}
		default:
			t.Errorf("gRPC Export(%q) received no request, want one exported span", protocol)
		}
	}
}
