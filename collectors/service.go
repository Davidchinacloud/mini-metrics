package collectors

import (
	"time"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	//"golang.org/x/net/context"
	//"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	//"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/tools/cache"
)

var (
	resyncPeriod = 10 * time.Minute
)

const (
	statusBuilding = iota
	statusFailed
	statusRunning
	statusStopped
)

type ServiceCollector struct {
	StatusBuilding		*prometheus.GaugeVec
	StatusFailed		*prometheus.GaugeVec
	StatusRunning		*prometheus.GaugeVec
	StatusStopped		*prometheus.GaugeVec
	pStore				podStore
	dStore				deploymentStore
	rStore      		replicasetStore
}

func RegisterServiceCollector(kubeClient kubernetes.Interface, namespace string) {
	podLister := registerPodCollector(kubeClient, namespace)
	dplLister := registerDeploymentCollector(kubeClient, namespace)
	replicaSetLister := registerReplicaSetCollector(kubeClient, namespace)

	prometheus.Register(newServiceCollector(podLister, dplLister, replicaSetLister))
}

func newServiceCollector(ps podStore, ds deploymentStore, rs replicasetStore)*ServiceCollector{
	labels := make(prometheus.Labels)

	return &ServiceCollector{
		StatusBuilding: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_building",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),
		StatusFailed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_failed",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),
		StatusRunning: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_runnning",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),
		StatusStopped: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_stopped",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),	
		pStore: ps,
		dStore: ds,
		rStore: rs,
	}
}

func (s *ServiceCollector) collectorList() []prometheus.Collector {
	return []prometheus.Collector{
		s.StatusBuilding,
		s.StatusFailed,
		s.StatusRunning,
		s.StatusStopped,
	}
}

func (s *ServiceCollector)calculateStatus(rs v1beta1.ReplicaSet)int{
	if rs.Status.AvailableReplicas == rs.Status.Replicas {
		return statusRunning
	}
	if rs.Status.ReadyReplicas < *rs.Spec.Replicas {
		return statusFailed
	}
	if *rs.Spec.Replicas == 0 {
		return statusStopped
	}
	return statusBuilding
}

func (s *ServiceCollector)calculateSetValue(){
	replicasets, err := s.rStore.List()
	if err != nil {
		glog.Errorf("listing replicasets failed: %s", err)
	} else {
		for _, r := range replicasets {
			status := s.calculateStatus(r)
			s.setStatus(r.Name, r.Namespace, status)
		}
	}
}

func (s *ServiceCollector)setStatus(name string, namespace string, status int){
	switch status {
		case statusBuilding:
			s.setValueBuilding(name, namespace)
		case statusRunning:
			s.setValueRunning(name, namespace)
		case statusFailed:
			s.setValueFailed(name, namespace)
		case statusStopped:
			s.setValueStopped(name, namespace)
		default:
			glog.Warningf("Unknow status: %d\n", status)
	}
}

func (s *ServiceCollector)setValueRunning(name string, namespace string){
	s.StatusRunning.WithLabelValues(name, namespace).Set(1)
	s.StatusBuilding.WithLabelValues(name, namespace).Set(0)
	s.StatusFailed.WithLabelValues(name, namespace).Set(0)
	s.StatusStopped.WithLabelValues(name, namespace).Set(0)
}

func (s *ServiceCollector)setValueFailed(name string, namespace string){
	s.StatusRunning.WithLabelValues(name, namespace).Set(0)
	s.StatusBuilding.WithLabelValues(name, namespace).Set(0)
	s.StatusFailed.WithLabelValues(name, namespace).Set(1)
	s.StatusStopped.WithLabelValues(name, namespace).Set(0)
}

func (s *ServiceCollector)setValueBuilding(name string, namespace string){
	s.StatusRunning.WithLabelValues(name, namespace).Set(0)
	s.StatusBuilding.WithLabelValues(name, namespace).Set(1)
	s.StatusFailed.WithLabelValues(name, namespace).Set(0)
	s.StatusStopped.WithLabelValues(name, namespace).Set(0)
}

func (s *ServiceCollector)setValueStopped(name string, namespace string){
	s.StatusRunning.WithLabelValues(name, namespace).Set(0)
	s.StatusBuilding.WithLabelValues(name, namespace).Set(0)
	s.StatusFailed.WithLabelValues(name, namespace).Set(0)
	s.StatusStopped.WithLabelValues(name, namespace).Set(1)
}

func (s *ServiceCollector)collect()error{
	glog.V(3).Infof("Collect at %v\n", time.Now())
	
	pods, err := s.pStore.List()
	if err != nil {
		glog.Errorf("listing pods failed: %s", err)
		return err
	} else {
		for _, pod := range pods {
			s.displayPod(pod)
		}
	}
	
	deployments, err := s.dStore.List()
	if err != nil {
		glog.Errorf("listing deployment failed: %s", err)
	} else {
		for _, d := range deployments {
			s.displayDeployment(d)
		}
	}
	
	replicasets, err := s.rStore.List()
	if err != nil {
		glog.Errorf("listing replicasets failed: %s", err)
	} else {
		for _, r := range replicasets {
			s.displayReplicaSet(r)
		}
	}
	
	s.calculateSetValue()
	
	return nil
}

func (s *ServiceCollector) Describe(ch chan<- *prometheus.Desc) {
	glog.V(3).Infof("Describe at %v\n", time.Now())
	for _, metric := range s.collectorList() {
		metric.Describe(ch)
	}
}

func (s *ServiceCollector) Collect(ch chan<- prometheus.Metric) {
	if err := s.collect(); err != nil {
		glog.Errorf("failed collecting service metrics: %v", err)
	}

	for _, metric := range s.collectorList() {
		metric.Collect(ch)
	}
}