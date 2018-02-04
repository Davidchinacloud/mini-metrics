package main

import (
	"fmt"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	//"github.com/sak0/mini-metrics/metrics"
	"github.com/sak0/mini-metrics/collectors"
)

const (
	healthzPath = "/healthz"
)

var (
	interval 	= flag.Duration("interval", 3 * time.Second, "How long collector interval.")
	port	 	= flag.String("port", "9090", "metrics listen port.")
	metricsPath = flag.String("metrics-path", "/metrics", "metrcis url path.")
)

type MiniExporter struct {
	mu	sync.Mutex
	collectors []prometheus.Collector
}

func NewMiniExporter()*MiniExporter{
	return &MiniExporter{
		collectors : []prometheus.Collector{
			collectors.NewServiceCollector(),
		},
	}
}

func (c *MiniExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, cc := range c.collectors {
		cc.Describe(ch)
	}
}

func (c *MiniExporter) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cc := range c.collectors {
		cc.Collect(ch)
	}
}

	
func main(){
	fmt.Printf("Mini Metrics Server.\n")
	defer fmt.Printf("Bye bye.\n")
	
	flag.Parse()
	//m := metrics.NewMetrics("123", *interval)
	//http.ListenAndServe("127.0.0.1:9090", m)
	
	err := prometheus.Register(NewMiniExporter())
	if err != nil {
		log.Fatalf("cannot export service error: %v", err)
	}
	
	
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Mini Exporter</title></head>
			<body>
			<h1>Mini Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
	})
	http.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	log.Fatal(http.ListenAndServe("0.0.0.0:9090", nil))
}