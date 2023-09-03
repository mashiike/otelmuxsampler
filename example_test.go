package otelmuxsampler_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mashiike/otelmuxsampler"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type onMemoryExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (e *onMemoryExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *onMemoryExporter) Shutdown(ctx context.Context) error {
	return nil
}

func Example() {
	exporter := &onMemoryExporter{}
	// only priority=high span is sampled
	mux := otelmuxsampler.Multiplexed(sdktrace.NeverSample())
	mux.AttributeEqual(attribute.String("priority", "high"), sdktrace.AlwaysSample())
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(mux)),
	)
	// operation1 is priority=high, operation2 is priority=low
	tr := tp.Tracer("example")
	func() {
		_, span := tr.Start(nil, "operation1", trace.WithAttributes(
			attribute.String("priority", "high"),
		))
		defer span.End()
	}()
	func() {
		_, span := tr.Start(nil, "operation2", trace.WithAttributes(
			attribute.String("priority", "low"),
		))
		defer span.End()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.Shutdown(ctx); err != nil {
		log.Fatal("failed to shutdown:", err)
	}

	// stdout to span names
	for _, span := range exporter.spans {
		fmt.Println(span.Name())
	}
	// Output:
	// operation1
}
