package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaeger "github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
)

type Message struct {
	Carrier opentracing.TextMapCarrier `json:"carrier"`
	Payload string                     `json:"message"`
}

func produce(tracer opentracing.Tracer, sc stan.Conn, event int) {
	m := Message{
		Payload: strconv.Itoa(event),
		Carrier: make(opentracing.TextMapCarrier),
	}

	span := tracer.StartSpan("SendEvent")
	span.SetTag("EventID", event)
	ext.SpanKindRPCClient.Set(span)
	span.Tracer().Inject(span.Context(), opentracing.TextMap, m.Carrier)

	toSend, err := json.Marshal(m)
	if err != nil {
		return
	}
	sc.PublishAsync("foo", toSend, func(ackedNuid string, err error) {

	})
	span.Finish()
}

func consume(tracer opentracing.Tracer, msg Message) {
	ctx, _ := tracer.Extract(opentracing.TextMap, msg.Carrier)
	span := tracer.StartSpan("Receiving", ext.RPCServerOption(ctx))
	span.SetTag("EventID", msg.Payload)

	<-time.After(500 * time.Microsecond)
	span.Finish()
	fmt.Printf("Received a message: %s\n", msg.Payload)
}

func producer(tracer opentracing.Tracer, sc stan.Conn, signalChan <-chan os.Signal) {
	ticker := time.NewTicker(3 * time.Second)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		count := 0
		for {
			select {
			case <-ticker.C:
				produce(tracer, sc, count)
				count++
			case <-signalChan:
				wg.Done()
				return
			}
		}
	}()
	wg.Wait()
}

func consumer(tracer opentracing.Tracer, sc stan.Conn, signalChan <-chan os.Signal) {
	sub, _ := sc.Subscribe("foo", func(m *stan.Msg) {
		var msg Message
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			fmt.Printf("Error in unmarshaling message")
		}
		consume(tracer, msg)
	})

	<-signalChan
	sub.Close()
}

func main() {
	natsURL := flag.String("nats", "", "the url of the nats streaming server")
	natsCluster := flag.String("cluster", "", "the id of the nats cluster we are connecting to")
	isProducer := flag.Bool("producer", false, "tells if the client starts with producer mode")
	clientName := flag.String("clientname", "", "Overrides the nats client name")

	flag.Parse()

	var name string

	switch {
	case *clientName != "":
		name = *clientName
	case *isProducer:
		name = "producer"
	default:
		name = "consumer"
	}

	// Setting up nats streaming connection
	sc, err := stan.Connect(*natsCluster, name, stan.NatsURL(*natsURL))
	if err != nil {
		log.Fatalf("Failed to connect %v", err)
	}
	defer sc.Close()

	// Setting up Jaeger as tracing collector
	tracer, closer := initJaeger(name)
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	// Signal chan to get interrupted
	signalChan := make(chan os.Signal, 0)
	signal.Notify(signalChan, os.Interrupt)

	// Produce / consume are blocking until a sigint is received
	if *isProducer {
		producer(tracer, sc, signalChan)
	} else {
		consumer(tracer, sc, signalChan)
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
