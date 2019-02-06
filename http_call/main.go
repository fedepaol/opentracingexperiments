package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

// Message represent a message to be sent on Nats Streaming
type Message struct {
	Payload string `json:"message"`
}

func produce(tracer opentracing.Tracer, URL string, event int) {
	span := tracer.StartSpan("SendEvent")
	defer span.Finish()

	span.SetTag("EventID", event)
	ext.SpanKindRPCClient.Set(span)

	msg := &Message{"Hello"}
	jsonVal, _ := json.Marshal(msg)

	req, _ := http.NewRequest("POST", URL, bytes.NewBuffer(jsonVal))
	// Inject the span context into the header
	span.Tracer().Inject(span.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(req.Header))

	if _, err := http.DefaultClient.Do(req); err != nil {
		span.SetTag("error", true)
		span.LogEvent(fmt.Sprintf("GET /async error: %v", err))
	}
}

func producer(tracer opentracing.Tracer, serverURL string) {
	ticker := time.NewTicker(3 * time.Second)

	count := 0
	for {
		select {
		case <-ticker.C:
			produce(tracer, serverURL, count)
			count++
			if count == 5 {
				return
			}

		}
	}

}

func handleRequests(tracer opentracing.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wireContext, _ := tracer.Extract(opentracing.TextMap, opentracing.HTTPHeadersCarrier(r.Header))
		sp := opentracing.StartSpan("Receiver", opentracing.ChildOf(wireContext))
		defer sp.Finish()
		fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
		time.Sleep(2 * time.Second)
	}
}

func consumer(tracer opentracing.Tracer) {
	http.HandleFunc("/", handleRequests(tracer))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	serverUrl := flag.String("server", "", "the url of the endpoint to call")
	isProducer := flag.Bool("producer", false, "tells if the client starts with producer mode")

	flag.Parse()

	var name string
	switch {
	case *isProducer:
		name = "producer"
	default:
		name = "consumer"
	}

	// Setting up Jaeger as tracing collector
	tracer, closer := initJaeger(name)
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	// Produce / consume are blocking until a sigint is received
	if *isProducer {
		producer(tracer, *serverUrl)
	} else {
		consumer(tracer)
	}
}

// initJaeger returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: "http://jaeger:14268/api/traces",
		},
	}
	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}
