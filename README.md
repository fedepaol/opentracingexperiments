## Opentracing + Nats streaming experiments

#### Opentracing for any api
When people speak about Opentracing, it often relates to http / rest calls. This is because it feels natural to embed / propagate our beloved Go context inside rest calls (the same applies to Grpc).

In this example, I wanted to see if / how we can leverage Opentracing with an Api that does not support context propagation, such as Nats Streaming.

There are four components in the `docker-compose.yml` file.

### The nats streaming server
Used to make the two endpoints talk to each other.

### Uber Jaeger
Used to collect data from tracing

#### A producer
The Go process that sends out messages

#### A consumer
The Go process that reads messages

### Span propagation
In order to link the span between the remote calls, the producer (and the consumer) use `Inject` and `Extract` as described [here](https://opentracing.io/docs/overview/inject-extract/)

The cool thing is that, as long as we use a supported (or a custom) `carrier`, the span can be linked between the calls.