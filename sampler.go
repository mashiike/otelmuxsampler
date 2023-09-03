package otelmuxsampler

import (
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Multiplexer is a Sampler that multiplexes sampling decisions
type Multiplexer struct {
	mu             sync.RWMutex
	defaultSampler sdktrace.Sampler
	lastIndex      int
	entries        []entry
	description    string
}

type entry struct {
	index   int
	name    string
	handler Handler
}

// Multiplexed returns a new Multiplexer
func Multiplexed(sampler sdktrace.Sampler) *Multiplexer {
	return &Multiplexer{
		defaultSampler: sampler,
	}
}

func (mux *Multiplexer) Handle(name string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.lastIndex++
	mux.entries = append(mux.entries, entry{
		index:   mux.lastIndex,
		name:    name,
		handler: handler,
	})

	var builder strings.Builder
	builder.WriteString("Multiplexer{")
	for _, e := range mux.entries {
		builder.WriteString(e.name)
		builder.WriteString(":")
		builder.WriteString(e.handler.Description())
		builder.WriteString(",")
	}
	builder.WriteString("default:")
	builder.WriteString(mux.defaultSampler.Description())
	builder.WriteString("}")
	mux.description = builder.String()
}

func (mux *Multiplexer) HandleFunc(name string, matcher func(sdktrace.SamplingParameters) bool, sampler sdktrace.Sampler) {
	mux.Handle(name, NewHandler(matcher, sampler))
}

func (mux *Multiplexer) AttributeEqual(attr attribute.KeyValue, sampler sdktrace.Sampler) {
	mux.Handle(
		fmt.Sprintf("attribute_equal(%s=%s)", attr.Key, attr.Value.Emit()),
		NewHandler(func(p sdktrace.SamplingParameters) bool {
			for _, a := range p.Attributes {
				if a.Key != attr.Key {
					continue
				}
				if a.Value.Emit() == attr.Value.Emit() {
					return true
				}
			}
			return false
		}, sampler),
	)
}

func (mux *Multiplexer) AttributeExists(attrKey attribute.Key, sampler sdktrace.Sampler) {
	mux.Handle(
		fmt.Sprintf("attribute_exists(%s)", attrKey),
		NewHandler(func(p sdktrace.SamplingParameters) bool {
			for _, a := range p.Attributes {
				if a.Key == attrKey {
					return true
				}
			}
			return false
		}, sampler),
	)
}

func (mux *Multiplexer) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	for _, e := range mux.entries {
		if e.handler.Match(p) {
			return e.handler.ShouldSample(p)
		}
	}
	return mux.defaultSampler.ShouldSample(p)
}

func (mux *Multiplexer) Description() string {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	return mux.description
}
