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
