package collectors

import (
	"fmt"
	"log"
	"time"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	resyncPeriod = 15 * time.Second
)


type PodLister func() ([]v1.Pod, error)
func (l PodLister) List() ([]v1.Pod, error) {
	return l()
}
type podStore interface {
	List() (pods []v1.Pod, err error)
}
type ServiceCollector struct {
	Status		*prometheus.GaugeVec
	store		podStore
}

func RegisterServiceCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface, namespace string) {
	client := kubeClient.CoreV1().RESTClient()
	glog.Infof("collect pod with %s", client.APIVersion())
	plw := cache.NewListWatchFromClient(client, "pods", namespace, fields.Everything())
	pinf := cache.NewSharedInformer(plw, &v1.Pod{}, resyncPeriod)

	podLister := PodLister(func() (pods []v1.Pod, err error) {
		for _, m := range pinf.GetStore().List() {
			pods = append(pods, *m.(*v1.Pod))
		}
		return pods, nil
	})

	//registry.MustRegister()
	fmt.Printf("just log for use var registry %v\n", registry)
	
	prometheus.Register(newServiceCollector(podLister))
	
	go pinf.Run(context.Background().Done())
}


func newServiceCollector(ps podStore)*ServiceCollector{
	labels := make(prometheus.Labels)

	return &ServiceCollector{
		Status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "fast_service_status",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service"},
		),
		store: ps,
	}
}

func (s *ServiceCollector) collectorList() []prometheus.Collector {
	return []prometheus.Collector{
		s.Status,
	}
}

func (s *ServiceCollector)collect()error{
	fmt.Printf("Collect at %v\n", time.Now())
	var status float64
	status = 1
	s.Status.WithLabelValues("node1234").Set(status)
	
	pods, err := s.store.List()
	if err != nil {
		glog.Errorf("listing pods failed: %s", err)
		return err
	}
	
	for _, pod := range pods {
		fmt.Printf("[pod] %v\n", pod.Status)
	}
	
	return nil
}

func (s *ServiceCollector) Describe(ch chan<- *prometheus.Desc) {
	fmt.Printf("Describe at %v\n", time.Now())
	for _, metric := range s.collectorList() {
		metric.Describe(ch)
	}
}

func (s *ServiceCollector) Collect(ch chan<- prometheus.Metric) {
	if err := s.collect(); err != nil {
		log.Println("failed collecting service metrics:", err)
	}

	for _, metric := range s.collectorList() {
		metric.Collect(ch)
	}
}

