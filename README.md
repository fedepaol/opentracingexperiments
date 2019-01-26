## Opentracing + Nats streaming experiments

#### Opentracing for any api
When people speak about Opentracing, it often relates to http / rest calls. This is because it feels natural to embed / propagate our beloved Go context inside rest calls (the same applies to Grpc).

In this example, I wanted to see if / how we can leverage Opentracing with an Api that does not support context propagation, such as Nats Streaming.

