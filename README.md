## Opentracing experiments

#### Opentracing for any api
When people speak about Opentracing, it often relates to http / rest calls. This is because it feels natural to embed / propagate our beloved span in http headers (the same applies to Grpc).

There are two examples here. One is the more traditional span propagation over http (a lot of examples can be found), the other by using a generic api (in this case nats streaming) in order to show that opentracing is able to propagate spans across any medium.



There are four kind of components in the `docker-compose.yml` file.

### The nats streaming server
Used to make the two endpoints talk to each other.

### Uber Jaeger
Used to collect data from tracing

### The nats streaming endpoints
Used to send / receive through nats

### The http endpoints
Used to send / receive through http

### Nats Span propagation
In this example, I wanted to see if / how we can leverage Opentracing with a generic Api such as Nats Streaming.

In order to link the span between the remote calls, the producer (and the consumer) use `Inject` and `Extract` as described [here](https://opentracing.io/docs/overview/inject-extract/)

The cool thing is that, as long as we use a supported (or a custom) `carrier`, the span can be linked between the calls.