## Introduction to OTEL (Open Telemetry)
Open Telemetry, often abbreviated as OTEL, stands as a versatile, open-source, and vendor-agnostic framework empowering developers and DevOps specialists with comprehensive tools for observing and monitoring applications. Covering a spectrum from logging to metrics and traces, Open Telemetry provides a robust foundation for gaining insights into application performance.

While logging is a familiar concept, metrics and traces may require clarification, especially for developers unfamiliar with the term "observability."

## Metrics: Unveiling Application Insights
Metrics in Open Telemetry involve aggregated information over the runtime of an application. This encompasses the ability to observe various aspects, such as the number of requests made to a server, the success or failure of requests, resource utilization, and even specifics like garbage collection (GC) metrics, heap usage, and CPU usage. The flexibility of metrics creation extends to anything observable within the application.

While some frameworks offer default metrics, they still need to be gathered and exported through a collector for monitoring on services like Prometheus.

## Traces: Navigating Request Lifecycles
Traces, a captivating component of observability, enable the visualization of a request's journey through services like Jaeger or Zipkin. Each request is assigned a unique traceId as the trace begins. Traces facilitate the tracking and observation of a call-chain of requests, akin to tracking a package's journey in a courier and delivery system.

Consider the following questions when analyzing a trace:

* How long did the request take?
* What entities were involved in handling the request?
* Which transport methods or vehicles were employed?

Traces allow for a granular understanding of breaks within the route, their duration, reasons for stops, and the frequency of stops. Open Telemetry's SDK and Trace Provider, along with the utilization of Spans, Span Events, and Span Records, play a crucial role in answering these questions.

### Baggage: Enhancing Contextual Information
In the context of Open Telemetry, baggage is akin to the context API in React. It serves as an object that can be propagated through the context to extract data relevant only to a specific request. While suitable for storing small values for tracing and logging, it is essential to avoid sending sensitive data through baggage.

The baggage serves as a valuable tool to enrich spans with detailed information, particularly accessible in specific services or layers of an application.

## Putting Knowledge into Practice: Hands-On Experience
To leverage Open Telemetry effectively, it's imperative to engage in hands-on activities. Setting up a Trace Provider, working with Spans, Span Events, and Span Records, and understanding the nuances of baggage usage contribute to a comprehensive and practical understanding of Open Telemetry in action. Dive into the world of Open Telemetry to enhance your application observability and monitoring capabilities.

While many programming languages provide robust support for Open Telemetry, this instance focuses on Golang. It's important to note that, in the current context, the logs SDK for Golang is not implemented. For future reference consult the list of [supported languages](https://opentelemetry.io/docs/languages/) and explore the [Open Telemetry repositories](https://github.com/orgs/open-telemetry/repositories). Always prioritize the [main repository](https://github.com/open-telemetry/opentelemetry-go) and its [contrib repository](https://github.com/open-telemetry/opentelemetry-go-contrib), housing extensions and instrumentation libraries crucial to the Open Telemetry framework. Stay updated with the latest developments to ensure seamless integration and enhanced functionality.

Before delving into hands-on activities, it's crucial to grasp the concept of a "Resource." In the context of Open Telemetry, a resource refers to metadata from our application that is added to the Tracer. Tracers, which are singleton objects, play a pivotal role in manipulating the Trace API and creating Spans.

The core idea behind observability is the effective propagation of context. Later, we will explore how to set up propagation to seamlessly gather data at different points throughout the lifespan of a request.

The following snippet, extracted from the [Otel's Golang documentation](https://opentelemetry.io/docs/languages/go/resources/), exemplifies the creation of resources in Golang:

```go
resources := resource.New(context.Background(),
    resource.WithFromEnv(), // Pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
    resource.WithProcess(), // Configure a set of Detectors that discover process information
    resource.WithOS(), // Configure a set of Detectors that discover OS information
    resource.WithContainer(), // Configure a set of Detectors that discover container information
    resource.WithHost(), // Configure a set of Detectors that discover host information
    resource.WithDetectors(thirdparty.Detector{}), // Bring your own external Detector implementation
    resource.WithAttributes(attribute.String("foo", "bar")), // Specify resource attributes directly
)
```

This data is then attached to the parent span:

![Span metadata containing information about the system and the sender](https://dev-to-uploads.s3.amazonaws.com/uploads/articles/o90oorx9hcc8mryvjbuu.png)



This code snippet showcases the flexibility of defining resources in Golang. It allows for pulling attributes from environment variables, configuring detectors for various types of information (process, OS, container, and host), and even incorporating custom external detectors. Additionally, specific resource attributes can be specified directly for a more tailored approach. Understanding and effectively utilizing resources is a fundamental step in maximizing the capabilities of Open Telemetry in your application.

### The setup

First, we proceed to set up telemetry helpers for our main server. Let's break down each part.

Before configuring the resource, we obtain the application context, which is the very first context as the application starts. We create our resource settings, which we'll later use when setting up the Tracer and Metric Providers.

```GO
ctx := context.Background()

res, err := resource.New(ctx,
    resource.WithFromEnv(),
    resource.WithProcess(),
    resource.WithTelemetrySDK(),
    resource.WithHost(),
    resource.WithAttributes(
        // the service name used to display traces in backends
        semconv.ServiceNameKey.String(serverName),
        attribute.String("environment", os.Getenv("GO_ENV")),
    ),
)

handleErr(err, "failed to create resource")
```

This is an optional step, but the Agent can be configured through an Environment Variable, dynamically setting the exporter address.
```GO
otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
if !ok {
    otelAgentAddr = "0.0.0.0:4317"
}
```

Now, we set up the metrics exporter and the metric provider. We specify that we are not using TLS, so the `WithInsecure` option from `otlptracegrpc` is passed as a configuration parameter.
```GO
metricExp, err := otlpmetricgrpc.New(
    ctx,
    otlpmetricgrpc.WithInsecure(),
    otlpmetricgrpc.WithEndpoint(otelAgentAddr),
)
handleErr(err, "Failed to create the collector metric exporter")

meterProvider := sdkmetric.NewMeterProvider(
    sdkmetric.WithResource(res),
    sdkmetric.WithReader(
        sdkmetric.NewPeriodicReader(
            metricExp,
            sdkmetric.WithInterval(2*time.Second),
        ),
    ),
)
otel.SetMeterProvider(meterProvider)
```

Finally, we set up the gRPC trace client to manage the exportation of our traces, create the exporter, set up a batch span processor, and instantiate our Trace Provider. We then set a propagation strategy and the trace provider.
```GO
traceClient := otlptracegrpc.NewClient(
    otlptracegrpc.WithInsecure(),
    otlptracegrpc.WithEndpoint(otelAgentAddr),
    otlptracegrpc.WithDialOption(grpc.WithBlock()),
)
traceExp, err := otlptrace.New(ctx, traceClient)
handleErr(err, "Failed to create the collector trace exporter")

bsp := sdktrace.NewBatchSpanProcessor(traceExp)
tracerProvider := sdktrace.NewTracerProvider(
    sdktrace.WithSampler(getSampler()),
    sdktrace.WithResource(res),
    sdktrace.WithSpanProcessor(bsp),
)

// set global propagator to tracecontext (the default is no-op).
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
otel.SetTracerProvider(tracerProvider)
```

All together create the following file

`otel.go`
```GO
package telemetry

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func InitProvider(serverName string) func() {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serverName),
			attribute.String("environment", os.Getenv("GO_ENV")),
		),
	)
		handleErr(err, "failed to create resource")

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	metricExp, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(otelAgentAddr))
	handleErr(err, "Failed to create the collector metric exporter")

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExp,
				sdkmetric.WithInterval(2*time.Second),
			),
		),
	)
	otel.SetMeterProvider(meterProvider)

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(getSampler()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := traceExp.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
		// pushes any last exports to the receiver
		if err := meterProvider.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

// Helper function to define sampling.
// When in development mode, AlwaysSample is defined,
// otherwise, sample based on Parent and IDRatio will be used.
func getSampler() sdktrace.Sampler {
	ENV := os.Getenv("GO_ENV")
	switch ENV {
	case "development":
		return sdktrace.AlwaysSample()
	case "production":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.5))
	default:
		return sdktrace.AlwaysSample()
	}
}

```


Now for our example server, we'll use the following code:

First, we instantiate our providers from the telemetry package, deferring the provider's cleanup function to be executed prior to stopping our program.
```GO
otelShutdown := telemetry.InitProvider(serverName)
defer otelShutdown()
```

On this program we'll be using the Gorilla Mux and the instrumentation package from the [opentelemetry-go-contrib](https://github.com/open-telemetry/opentelemetry-go-contrib) otelmux. Explained in detail, after instantiating the router we are adding the otelmux middleware for adding the option which formats the Span Name with the method and the route used.

In this program, we utilize the Gorilla Mux and the instrumentation package from [opentelemetry-go-contrib](https://github.com/open-telemetry/opentelemetry-go-contrib) (otelmux). After instantiating the router, we add the otelmux middleware to format the Span Name with the method and the route used.
```GO
router := mux.NewRouter()
router.Use(
	otelmux.Middleware(
		serverName,
		otelmux.WithSpanNameFormatter(func(routeName string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, routeName)
		}),
	),
)
```

Moving forward to our handler, streamlined with a few objects and methods from the OpenTelemetry framework. We leverage the baggage package to obtain the baggage propagated through the context. After obtaining the span from the context, we use the getter method Member from the baggage package to retrieve the values of destination and transportation (defined initially). We then create trace attributes for them and add them to a Span Event.
```GO
router.HandleFunc("/packages/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	// package response
	pr := getPackage(r.Context(), id)

	baggage := baggage.FromContext(r.Context())

	// late acquisition of the span to add attributes
	span := trace.SpanFromContext(r.Context())
	destination := baggage.Member("destination").Value()
	transportation := baggage.Member("transportation").Value()
	destinationAttr := trace.WithAttributes(attribute.String("destination", destination))
	transportationAttr := trace.WithAttributes(attribute.String("transportation", transportation))
	span.AddEvent("Obtaining package", destinationAttr, transportationAttr)

	reply := fmt.Sprintf("package is %s (id %s)\n", pr, id)
	_, _ = w.Write(([]byte)(reply))
})
```

At last, we define the `getPackage` method, taking the context as the first argument. The context must be propagated at all times, not just for handling cancellation, but also playing a key role in telemetry. We instantiate a new child span from our Provider and the current context. We defer its cleanup function when we call `defer span.End()` to ensure no resources are leaked. We create a Span Event named `getPackage` and add the trace attribute for the package and the id. If the package is found, we add another event; if it fails, we record the exception within the span.

```GO
func getPackage(ctx context.Context, id string) string {
	_, span := trace.SpanFromContext(ctx).TracerProvider().Tracer(serverName).Start(ctx, "getPackage")
	defer span.End()

	span.AddEvent("getPackage", trace.WithAttributes(attribute.String("package", id)))
	if id == "123" {
		span.AddEvent("found package")
		return "found package"
	}
	span.RecordError(fmt.Errorf("package not found"))
	return "unknown"
}
```

All put together, it creates the following file:

`main.go`
```GO
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/sosalejandro/otel-example/commons/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

const serverName = "otel-example-server"

func main() {
	// ...

	otelShutdown := telemetry.InitProvider(serverName)
	defer otelShutdown()

	router := mux.NewRouter()
	router.Use(
		otelmux.Middleware(
			serverName,
			otelmux.WithSpanNameFormatter(func(routeName string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, routeName)
			})),
	)

	router.HandleFunc("/packages/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		// package response
		pr := getPackage(r.Context(), id)

		baggage := baggage.FromContext(r.Context())

		// late adquisition of the span to add attributes
		span := trace.SpanFromContext(r.Context())
		destination := baggage.Member("destination").Value()
		transportation := baggage.Member("transportation").Value()
		destinationAttr := trace.WithAttributes(attribute.String("destination", destination))
		transportationAttr := trace.WithAttributes(attribute.String("transportation", transportation))
		span.AddEvent("Obtaining package", destinationAttr, transportationAttr)

		reply := fmt.Sprintf("package is %s (id %s)\n", pr, id)
		_, _ = w.Write(([]byte)(reply))
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	if err := runServer(server); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func runServer(server *http.Server) error {
	// Start the server in a separate goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for an interrupt signal to gracefully shut down the server
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Println("Shutting down server...")

	// Create a context with a timeout of 5 seconds to allow outstanding requests to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shut down the server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
		return err
	}

	log.Println("Server shut down.")

	return nil
}

func getPackage(ctx context.Context, id string) string {
	_, span := trace.SpanFromContext(ctx).TracerProvider().Tracer(serverName).Start(ctx, "getPackage")
	defer span.End()

	span.AddEvent("getPackage", trace.WithAttributes(attribute.String("package", id)))
	if id == "123" {
		span.AddEvent("found package")
		return "found package"
	}
	span.RecordError(fmt.Errorf("package not found"))
	return "unknown"
}

```

Now, for the client application, we define a simple `initTracer` within the `main.go` file for our client application:

```GO
func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	res, _ := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serverName),
			attribute.String("environment", os.Getenv("GO_ENV")),
		),
	)

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")

	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	traceExp, err := otlptrace.New(ctx, traceClient)
	telemetry.HandleErr(err, "Failed to create the collector trace exporter")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(telemetry.GetSampler()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider, nil
}
```

Moving forward to the main function, we declare our tracer and a cleanup function:

```GO
tp, err := initTracer()
if err != nil {
	log.Fatal(err)
}
defer func() {
	if err := tp.Shutdown(context.Background()); err != nil {
		telemetry.HandleErr(err, "Error shutting down tracer provider")
	}
}()
```

Next, we create our URL and an HTTP client using the `otelhttp` instrumentation package and its Transport:
```GO
url := flag.String("server", "http://localhost:8080/packages/123", "server url")
flag.Parse()

client := http.Client{
	Transport: otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
			return otelhttptrace.NewClientTrace(ctx)
		}),
	),
}
```

Now, we set the baggage values and add it to the context:
```GO
bag, _ := baggage.Parse("destination=newyork,transportation=truck")
ctx := baggage.ContextWithBaggage(context.Background(), bag)
```

Finally, we define our body variable, obtain the tracer, and create an anonymous function, which will be our handler in this case. We start our span, define its name, and add the trace attribute using the semantic convention for Peer Service, declaring otel-example-server as our peer service. We defer the span and create the request:

```GO
var body []byte

tr := otel.Tracer(serverName)
err = func(ctx context.Context) error {
	ctx, span := tr.Start(
		ctx,
		"Otel propagation example: sending package from Boston",
		trace.WithAttributes(semconv.PeerService("otel-example-client")))
	defer span.End()
	req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)

	fmt.Printf("Sending request...\n")
	span.AddEvent("Sending request...")
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err = io.ReadAll(res.Body)
	span.AddEvent("Request received")
	_ = res.Body.Close()

	return err
}(ctx)
```

At last, we handle the error and wait for 10 seconds while the spans are exported:

```GO
if err != nil {
	telemetry.HandleErr(err, "Error executing handler request")
}

fmt.Printf("Response Received: %s\n\n\n", body)
fmt.Printf("Waiting for a few seconds to export spans ...\n\n")
time.Sleep(10 * time.Second)
fmt.Printf("Inspect traces on Jaeger\n")
```

Everything put together within the `main.go` looks like this:

```GO
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"

	"github.com/sosalejandro/otel-example/commons/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const serverName = "otel-example-client"

func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	res, _ := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serverName),
			attribute.String("environment", os.Getenv("GO_ENV")),
		),
	)

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")

	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	traceExp, err := otlptrace.New(ctx, traceClient)
	telemetry.HandleErr(err, "Failed to create the collector trace exporter")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(telemetry.GetSampler()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider, nil
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			telemetry.HandleErr(err, "Error shutting down tracer provider")
		}
	}()

	url := flag.String("server", "http://localhost:8080/packages/123", "server url")
	flag.Parse()

	client := http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			}),
		),
	}

	bag, _ := baggage.Parse("destination=newyork,transportation=truck")
	ctx := baggage.ContextWithBaggage(context.Background(), bag)

	var body []byte

	tr := otel.Tracer(serverName)
	err = func(ctx context.Context) error {
		ctx, span := tr.Start(
			ctx,
			"Otel propagation example: sending package from boston",
			trace.WithAttributes(semconv.PeerService("otel-example-server")))
		defer span.End()
		req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)

		span.AddEvent("Sending request...")
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		body, err = io.ReadAll(res.Body)
		span.AddEvent("Request received")
		_ = res.Body.Close()

		return err
	}(ctx)

	if err != nil {
		telemetry.HandleErr(err, "Error executing handler request")
	}

	fmt.Printf("Response Received: %s\n\n\n", body)
	fmt.Printf("Waiting for few seconds to export spans ...\n\n")
	time.Sleep(10 * time.Second)
	fmt.Printf("Inspect traces on jaeger\n")
}
```

### The Containers Setup

`jaeger-ui.json`
```JSON
{
    "monitor": {
      "menuEnabled": true
    },
    "dependencies": {
      "menuEnabled": true
    }
}
```

`otel-collector-config.yml`
```YML
receivers:
  otlp:
    protocols:
      grpc:

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"

  logging:

  zipkin:
    endpoint: "http://zipkin-all-in-one:9411/api/v2/spans"
    format: proto

  otlp:
    endpoint: jaeger-all-in-one:4317
    tls:
      insecure: true

processors:
  batch:

extensions:
  health_check:
  pprof:
    endpoint: :1888
  zpages:
    endpoint: :55679

service:
  extensions: [pprof, zpages, health_check]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, zipkin, otlp]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, prometheus]
```

`prometheus.yml`
```YAML
scrape_configs:
  - job_name: 'otel-collector'
    scrape_interval: 10s
    static_configs:
      - targets: ['otel-collector:8889']
      - targets: ['otel-collector:8888']
```

`docker-compose.yml`
```YAML
version: '3'

services:
  # Jaeger
  jaeger-all-in-one:
    image: jaegertracing/all-in-one:latest
    restart: always
    ports:
      - "16686:16686"
      - "14268"
      - "14250"

  # Zipkin
  zipkin-all-in-one:
    image: openzipkin/zipkin:latest
    restart: always
    ports:
      - "9411:9411"

  # Collector
  otel-collector:
    image: otel/opentelemetry-collector:latest
    restart: always
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yml:/etc/otel-collector-config.yaml
    ports:
      - "1888:1888"   # pprof extension
      - "8888:8888"   # Prometheus metrics exposed by the collector
      - "8889:8889"   # Prometheus exporter metrics
      - "13133:13133" # health_check extension
      - "4317:4317"   # OTLP gRPC receiver
      - "55679:55679" # zpages extension
    depends_on:
      - jaeger-all-in-one
      - zipkin-all-in-one

  prometheus:
    container_name: prometheus
    image: prom/prometheus:latest
    restart: always
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
```

At last, a Makefile to help us setup and clean things easily:

```Makefile
# Build stage
build:
	@echo "Creating docker compose..."
	docker compose create
	@echo "Building server app..."
	go build -o server_app ./app1/main.go 
	@echo "Building client app..."
	go build -o client_app ./app2/main.go 
	@echo "Build stage completed."

setup:
	@echo "Setting up docker compose..."
	docker compose up -d
	@echo "Setting up server app..."
	./server_app & echo $$! > server_app.pid
	@echo "Setup stage completed."

run:
	@echo "Running client app..."
	./client_app 
	@echo "Run stage completed."
	
clean:
	@echo "Cleaning up..."
	docker compose down
	@echo "Cleaning up server app..."
	kill `cat server_app.pid`
	rm -f server_app server_app.pid
	@echo "Cleaning up client app..."
	rm -f client_app
	@echo "Clean stage completed."
```

We'll wrap-up with a demostration of the following code when ran and how it looks on either jaeger and zipkin, following with the metrics for a next episode. 