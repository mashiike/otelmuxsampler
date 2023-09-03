package otelmuxsampler_test

import (
	"fmt"
	"testing"

	"github.com/mashiike/otelmuxsampler"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func decisionToString(d sdktrace.SamplingDecision) string {
	switch d {
	case sdktrace.RecordAndSample:
		return "RecordAndSample"
	case sdktrace.RecordOnly:
		return "RecordOnly"
	case sdktrace.Drop:
		return "Drop"
	default:
		return "Unknown"
	}
}

func TestMultiplexerAttributeEqual(t *testing.T) {
	mux := otelmuxsampler.Multiplexed(sdktrace.NeverSample())
	mux.AttributeEqual(attribute.String("server.request.class", "critical"), sdktrace.AlwaysSample())
	mux.AttributeEqual(attribute.String("server.request.class", "high"), sdktrace.NeverSample())
	mux.AttributeEqual(attribute.String("server.request.class", "medium"), sdktrace.NeverSample())
	mux.AttributeEqual(attribute.String("server.request.class", "low"), sdktrace.NeverSample())

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")

	t.Run("critical", func(t *testing.T) {
		params := sdktrace.SamplingParameters{
			TraceID: traceID,
			Attributes: []attribute.KeyValue{
				attribute.String("server.request.class", "critical"),
			},
		}
		actualDecision := mux.ShouldSample(params).Decision
		expectedDecision := sdktrace.AlwaysSample().ShouldSample(params).Decision
		if actualDecision != expectedDecision {
			t.Errorf("ShouldSample should be %s, got %s instead",
				decisionToString(expectedDecision),
				decisionToString(actualDecision),
			)
		}
	})
	t.Run("low", func(t *testing.T) {
		params := sdktrace.SamplingParameters{
			TraceID: traceID,
			Attributes: []attribute.KeyValue{
				attribute.String("server.request.class", "low"),
			},
		}
		actualDecision := mux.ShouldSample(params).Decision
		expectedDecision := sdktrace.NeverSample().ShouldSample(params).Decision
		if actualDecision != expectedDecision {
			t.Errorf("ShouldSample should be %s, got %s instead",
				decisionToString(expectedDecision),
				decisionToString(actualDecision),
			)
		}
	})
}

func TestMultiplexerDescriptoin(t *testing.T) {
	mux := otelmuxsampler.Multiplexed(sdktrace.AlwaysSample())
	mux.Handle("internal_ignore", otelmuxsampler.NewHandler(
		func(p sdktrace.SamplingParameters) bool {
			return p.Kind == trace.SpanKindInternal
		},
		sdktrace.NeverSample(),
	))
	mux.Handle("server_always", otelmuxsampler.NewHandler(
		func(p sdktrace.SamplingParameters) bool {
			return p.Kind == trace.SpanKindServer
		},
		sdktrace.AlwaysSample(),
	))
	expectedDescription := fmt.Sprintf(
		"Multiplexer{internal_ignore:%s,server_always:%s,default:%s}",
		sdktrace.NeverSample().Description(),
		sdktrace.AlwaysSample().Description(),
		sdktrace.AlwaysSample().Description(),
	)

	if mux.Description() != expectedDescription {
		t.Errorf("Sampler description should be %s, got '%s' instead",
			expectedDescription,
			mux.Description(),
		)
	}
}
