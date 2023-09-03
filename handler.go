package otelmuxsampler

import sdktrace "go.opentelemetry.io/otel/sdk/trace"

type Handler interface {
	Match(sdktrace.SamplingParameters) bool
	sdktrace.Sampler
}

func NewHandler(matcher func(sdktrace.SamplingParameters) bool, sampler sdktrace.Sampler) Handler {
	return &handler{
		matcher: matcher,
		Sampler: sampler,
	}
}

type handler struct {
	sdktrace.Sampler
	matcher func(sdktrace.SamplingParameters) bool
}

func (h *handler) Match(p sdktrace.SamplingParameters) bool {
	return h.matcher(p)
}
