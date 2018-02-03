package main

import (
	"fmt"
	"net/http"
	"flag"
	"time"
	
	"github.com/sak0/mini-metrics/metrics"
)

var interval = flag.Duration("interval", 3 * time.Second, "How long collector interval.")

func main(){
	fmt.Printf("Mini Metrics Server.\n")
	defer fmt.Printf("Bye bye.\n")
	
	flag.Parse()
	m := metrics.NewMetrics("123", *interval)
	http.ListenAndServe("127.0.0.1:9090", m)
}

