package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
)

func main() {
	natsURL := flag.String("nats", "", "the url of the nats streaming server")
	natsCluster := flag.String("cluster", "", "the id of the nats cluster we are connecting to")
	producer := flag.Bool("producer", true, "tells if the client starts with producer mode")

	flag.Parse()

	var name string
	if *producer {
		name = "producer"
	} else {
		name = "consumer"
	}
	sc, err := stan.Connect(*natsCluster, name, stan.NatsURL(*natsURL))
	if err != nil {
		log.Fatalf("Failed to connect %v", err)
	}
	defer sc.Close()

	signalChan := make(chan os.Signal, 0)
	signal.Notify(signalChan, os.Interrupt)

	if *producer {
		produce(sc, signalChan)
	} else {
		consume(sc, signalChan)
	}
}

func produce(sc stan.Conn, signalChan <-chan os.Signal) {
	ticker := time.NewTicker(time.Second)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Println("publishing")
				sc.Publish("foo", []byte("Hello World"))
			case <-signalChan:
				wg.Done()
				return
			}
		}

	}()
	wg.Wait()
}

func consume(sc stan.Conn, signalChan <-chan os.Signal) {
	sub, _ := sc.Subscribe("foo", func(m *stan.Msg) {
		fmt.Printf("Received a message: %s\n", string(m.Data))
	})

	<-signalChan
	sub.Close()
}
