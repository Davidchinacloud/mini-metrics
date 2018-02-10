package main

import (
	"fmt"
	"flag"
	"log"
	"net"
	"net/http"
	"time"
	"strconv"
	
	"github.com/golang/glog"
	clientset "k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	
	"github.com/prometheus/client_golang/prometheus/promhttp"
	//"github.com/sak0/mini-metrics/metrics"
	"github.com/sak0/mini-metrics/collectors"
)

const (
	healthzPath = "/healthz"
)

var (
	interval 	= flag.Duration("interval", 3 * time.Second, "How long collector interval.")
	port	 	= flag.Int("port", 8080, "metrics listen port.")
	metricsPath = flag.String("metrics-path", "/metrics", "metrcis url path.")
	namespace	= flag.String("namespace", metav1.NamespaceAll, "namespace to be enabled for monitoring")
	
	defaultCollectors = []string{"services"}
	availableCollectors = map[string]func(kubeClient clientset.Interface, namespace string){
		"services":                 collectors.RegisterServiceCollector,
	}	
)

func registerCollectors(kubeClient clientset.Interface, collectors []string, namespace string){
		for _, c := range collectors{
			if f, ok := availableCollectors[c]; !ok {
				glog.Warningf("Collector %s is not available", c)
			} else {
				f(kubeClient, namespace)
			}
		}
	}

func createKubenetesClient()(kubeClient clientset.Interface, err error){
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	if kubeClient, err = clientset.NewForConfig(config); err != nil {
		return nil, err
	}
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}
	glog.Infof("Running with Kubernetes cluster version: v%s.%s. git version: %s. git tree state: %s. commit: %s. platform: %s",
		v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)
	return kubeClient, nil
}
	
func main(){
	fmt.Printf("Mini Metrics Server.\n")
	defer fmt.Printf("Bye bye.\n")
	
	flag.Parse()
	//m := metrics.NewMetrics("123", *interval)
	//http.ListenAndServe("127.0.0.1:9090", m)
	
	kubeClient, err := createKubenetesClient()
	if err != nil {
		glog.Errorf("Can't create kubeneres client: %v\n", err)
		return
	}
	
	registerCollectors(kubeClient, defaultCollectors, *namespace)
	
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
	
	listenAddress := net.JoinHostPort("0.0.0.0", strconv.Itoa(*port))
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}