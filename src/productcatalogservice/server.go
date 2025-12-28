package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/honeycombio/microservices-demo/src/productcatalogservice/demo/msdemo"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	cat pb.ListProductsResponse
	log *logrus.Logger

	db   *sqlx.DB
	port = "3550"

	reloadCatalog bool
)

func init() {
	log = logrus.New()
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

func initOtelTracing(ctx context.Context, log logrus.FieldLogger) *sdktrace.TracerProvider {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "opentelemetry-collector:4317"
	}

	// Set GRPC options to establish an insecure connection to an OpenTelemetry Collector
	// To establish a TLS connection to a secured endpoint use:
	//   otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	}

	// Create the exporter
	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	if err != nil {
		log.Fatal(err)
	}

	// Specify the TextMapPropagator to ensure spans propagate across service boundaries
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}))

	// Set standard attributes per semantic conventions
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("productcatalog"),
	)

	// Create and set the TraceProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp
}

func main() {
	// Initialize OpenTelemetry Tracing
	ctx := context.Background()
	tp := initOtelTracing(ctx, log)
	defer func() { _ = tp.Shutdown(ctx) }()

	flag.Parse()

	var err error

	db, err = sqlx.Open("mysql",
		fmt.Sprintf("root:root@(%s:%s)/productcatalogservice",
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT")))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Connect and check the server version
	var version string
	db.QueryRow("SELECT VERSION()").Scan(&version)
	fmt.Println("Connected to:", version)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for {
			sig := <-sigs
			log.Printf("Received signal: %s", sig)
			if sig == syscall.SIGUSR1 {
				reloadCatalog = true
				log.Infof("Enable catalog reloading")
			} else {
				reloadCatalog = false
				log.Infof("Disable catalog reloading")
			}
		}
	}()

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Infof("starting grpc server at :%s", port)
	run(port)
	select {}
}

func run(port string) string {
	l, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	// create gRPC server with OpenTelemetry instrumentation on all incoming requests
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
	)

	svc := &productCatalog{}
	pb.RegisterProductCatalogServiceServer(srv, svc)
	healthpb.RegisterHealthServer(srv, svc)
	go func() {
		_ = srv.Serve(l)
	}()
	return l.Addr().String()
}

type productCatalog struct{}

func getRandomWaitTime(max int, buckets int) float32 {
	num := float32(0)
	val := float32(max / buckets)
	for i := 0; i < buckets; i++ {
		num += rand.Float32() * val
	}
	return num
}

func sleepRandom(max int) {
	rnd := getRandomWaitTime(max, 4)
	time.Sleep((time.Duration(rnd)) * time.Millisecond)
}

func (p *productCatalog) Check(_ context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *productCatalog) Watch(_ *healthpb.HealthCheckRequest, _ healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *productCatalog) ListProducts(ctx context.Context, _ *pb.Empty) (*pb.ListProductsResponse, error) {
	var products []*pb.Product
	rows, err := db.QueryxContext(ctx, "SELECT * FROM products")
	if err != nil {
		log.Fatalln(err)
	}
	for rows.Next() {
		products = append(products, p.mustTransformRows(rows))
	}
	return &pb.ListProductsResponse{Products: products}, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("app.product_id", req.GetId()))

	rows, err := db.QueryxContext(ctx, "SELECT * FROM products WHERE id = ?", req.GetId())
	if err != nil {
		log.Fatalln(err)
	}

	if rows.Next() {
		return p.mustTransformRows(rows), nil
	} else {
		return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
	}
}

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	var ps []*pb.Product
	rows, err := db.QueryxContext(ctx, "SELECT * FROM products WHERE name LIKE ? OR description LIKE ?", req.Query, req.Query)
	if err != nil {
		log.Fatalln(err)
	}
	for rows.Next() {
		ps = append(ps, p.mustTransformRows(rows))
	}
	return &pb.SearchProductsResponse{Results: ps}, nil
}

func (p *productCatalog) mustTransformRows(rows *sqlx.Rows) *pb.Product {
	var product struct {
		Id          string `db:"id,omitempty"`
		Name        string `db:"name,omitempty"`
		Description string `db:"description,omitempty"`
		Picture     string `db:"picture,omitempty"`
		PriceUsd    []byte `db:"price_usd,omitempty"`
		Categories  []byte `db:"categories,omitempty"`
	}

	err := rows.StructScan(&product)
	if err != nil {
		log.Fatalln(err)
	}

	var result pb.Product
	var money pb.Money

	result.Id = product.Id
	result.Name = product.Name
	result.Description = product.Description
	result.Picture = product.Picture

	var categories []string
	if err := json.Unmarshal(product.PriceUsd, &money); err != nil {
		log.Fatalln(err)
	}
	if err := json.Unmarshal(product.Categories, &categories); err != nil {
		log.Fatalln(err)
	}

	result.PriceUsd = &money
	result.Categories = categories
	return &result
}
