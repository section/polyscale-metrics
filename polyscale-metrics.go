package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"time"

	"github.com/jackc/pgx/v5"
)

var (
	CacheLatenciesVec = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "polyscale_metrics_latency_cache",
		Help:       "The latency of queries to the cache",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"popname"})
	OriginLatenciesVec = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "polyscale_metrics_latency_origin",
		Help:       "The latency of queries to the origin",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"popname"})
)

var CacheConn, OriginConn *pgx.Conn

func dbSetup() {
	urlExample := os.Getenv("CACHE_DATABASE_URL")
	var err error
	CacheConn, err = pgx.Connect(context.Background(), urlExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to cache database: %v\n", err)
		panic(0)
	}

	urlExample = os.Getenv("ORIGIN_DATABASE_URL")
	OriginConn, err = pgx.Connect(context.Background(), urlExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to origin database: %v\n", err)
		panic(0)
	}
}

func dbQuery(pop string, latenciesVec *prometheus.SummaryVec, conn *pgx.Conn, which string, query string) {
	// Time the query
	start := time.Now()
	rows, err := conn.Query(context.Background(), query)
	duration := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		panic(0)
	} else {
		// Log the time in prometheus, only for successful queries
		latencies := latenciesVec.WithLabelValues(pop)
		latencies.Observe(float64(duration.Milliseconds()))
	}

	// If we don't scan the row, future queries will be 0ms
	scan(rows)

	fmt.Print(which, " ", float64(duration.Milliseconds()), "ms ")
}

func scan(rows pgx.Rows) {
	var err error

	for rows.Next() {
		var v []interface{}
		v, err = rows.Values()
		_ = v
		if err != nil {
			fmt.Fprintf(os.Stderr, "Values failed: %v\n", err)
			panic(0)
		}
	}
}

func recordMetricsForever(pop string, intervalSeconds int, query string) {
	for {
		fmt.Print("nodename ", pop, " ")
		dbQuery(pop, CacheLatenciesVec, CacheConn, "cache", query)
		dbQuery(pop, OriginLatenciesVec, OriginConn, "origin", query)
		fmt.Println()
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
}

func main() {
	metricsPort := ":2112"
	intervalSeconds := 60 // Default

	nodeName := os.Getenv("NODE_NAME") // Node name provided by deployment yaml
	if nodeName == "" {
		nodeName = "local"
	}

	interval := os.Getenv("INTERVAL") // Seconds between queries
	if interval != "" {
		var err error
		intervalSeconds, err = strconv.Atoi(interval)
		if err != nil {
			intervalSeconds = 60
		}
	}

	query := os.Getenv("QUERY")
	if query == "" {
		fmt.Fprintf(os.Stderr, "QUERY needs to be specified as an environment variable.\n")
		panic(0)
	}

	metricsEndpoint := "/metrics"
	fmt.Println("Node", nodeName, "listening to", metricsEndpoint, "on port", metricsPort, "interval", intervalSeconds, "s", "query", query)

	dbSetup()
	go recordMetricsForever(nodeName, intervalSeconds, query)
	http.Handle(metricsEndpoint, promhttp.Handler())
	http.ListenAndServe(metricsPort, nil)
}
