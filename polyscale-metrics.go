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
	myCounter           = 0
	queriesProcessedVec = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "polyscale_metrics_processed_queries_total",
		Help: "The total number of processed queries",
	}, []string{"popname"})
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
		os.Exit(1)
	}

	urlExample = os.Getenv("ORIGIN_DATABASE_URL")
	OriginConn, err = pgx.Connect(context.Background(), urlExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to origin database: %v\n", err)
		os.Exit(1)
	}
}

func dbQuery(pop string, latenciesVec *prometheus.SummaryVec, conn *pgx.Conn, which string) {
	var orderdate, region, city, category string
	var product string
	var qty int64
	var unitprice, totalprice float64

	// Time the query
	start := time.Now()
	rows, err := conn.Query(context.Background(), "select * from foodsales limit 1;")
	duration := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	} else {
		// Log the time in prometheus, only for successful queries
		latencies := latenciesVec.WithLabelValues(pop)
		latencies.Observe(float64(duration.Milliseconds()))
	}

	// If we don't scan the row, future queries will be 0ms
	err, _ = scanRecord(rows, err, orderdate, region, city, category, product, qty, unitprice, totalprice)

	fmt.Print(which, " ", float64(duration.Milliseconds()), "ms ")
}

func scanRecord(rows pgx.Rows, err error, orderdate string, region string, city string, category string, product string, qty int64, unitprice float64, totalprice float64) (error, bool) {
	for rows.Next() {
		err = rows.Scan(&orderdate, &region, &city, &category, &product, &qty, &unitprice, &totalprice)
		if err != nil {
			fmt.Printf("Scan error: %v", err)
			return nil, true
		}
	}

	return err, false
}

func recordMetricsForever(pop string, intervalSeconds int) {
	queriesProcessed := queriesProcessedVec.WithLabelValues(pop)
	for {
		queriesProcessed.Inc()
		myCounter++
		fmt.Print("nodename ", pop, " ")
		dbQuery(pop, CacheLatenciesVec, CacheConn, "cache")
		dbQuery(pop, OriginLatenciesVec, OriginConn, "origin")
		fmt.Println()
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
}

func main() {
	metricsPort := ":2112"
	intervalSeconds := 60 // Default

	dbSetup()

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
	metricsEndpoint := "/metrics"
	go recordMetricsForever(nodeName, intervalSeconds)
	http.Handle(metricsEndpoint, promhttp.Handler())
	fmt.Println("Node", nodeName, "listening to", metricsEndpoint, "on port", metricsPort, "interval", intervalSeconds, "s")
	http.ListenAndServe(metricsPort, nil)
}
