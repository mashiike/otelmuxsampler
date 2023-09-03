package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/mashiike/otelmuxsampler"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	cleanup, err := setupTraceProvider()
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("/critical", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("critical"))
	})
	mux.HandleFunc("/high", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("high"))
	})
	mux.HandleFunc("/medium", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("medium"))
	})
	mux.HandleFunc("/low", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("low"))
	})
	paths := []string{"/critical", "/high", "/medium", "/low"}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		redirectTo := paths[rnd.Intn(len(paths))]
		http.Redirect(w, r, redirectTo, http.StatusFound)
	})
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("[info] request:", r.URL.Path)
			otelhttp.NewHandler(otelhttp.WithRouteTag(r.URL.Path, next), r.URL.Path, otelhttp.WithSpanOptions(
				trace.WithAttributes(attribute.String("server.request.class", getRequestClass(r))),
			)).ServeHTTP(w, r)
		})
	}
	log.Println("[info] start server, access to http://localhost:8080/")
	if err := http.ListenAndServe(":8080", middleware(mux)); err != nil {
		log.Fatal("[error]", err)
	}
}

func getRequestClass(r *http.Request) string {
	switch r.URL.Path {
	case "/critical":
		return "critical"
	case "/high":
		return "high"
	case "/medium":
		return "medium"
	default:
		return "low"
	}
}

func setupTraceProvider() (func(), error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(os.Stdout),
	)
	if err != nil {
		return func() {}, fmt.Errorf("failed to create stdout exporter: %w", err)
	}
	mux := otelmuxsampler.Multiplexed(sdktrace.NeverSample())
	mux.AttributeEqual(attribute.String("server.request.class", "critical"), sdktrace.AlwaysSample())
	mux.AttributeEqual(attribute.String("server.request.class", "high"), sdktrace.TraceIDRatioBased(0.5))
	mux.AttributeEqual(attribute.String("server.request.class", "medium"), sdktrace.TraceIDRatioBased(0.01))
	mux.AttributeEqual(attribute.String("server.request.class", "low"), sdktrace.NeverSample())

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(mux)),
	)
	otel.SetTracerProvider(tp)
	log.Println("[info] setup trace provider")
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Println("[error] failed to shutdown:", err)
		}
	}, nil
}
