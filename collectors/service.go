package collectors

import (
	"fmt"
	"log"
	"time"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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

type PodLister func() ([]v1.Pod, error)
func (l PodLister) List() ([]v1.Pod, error) {
	return l()
}
type podStore interface {
	List() (pods []v1.Pod, err error)
}

type DeploymentLister func() ([]v1beta1.Deployment, error)
func (l DeploymentLister) List() ([]v1beta1.Deployment, error) {
	return l()
}
type deploymentStore interface {
	List() (deployments []v1beta1.Deployment, err error)
}

type ReplicaSetLister func() ([]v1beta1.ReplicaSet, error)
func (l ReplicaSetLister) List() ([]v1beta1.ReplicaSet, error) {
	return l()
}
type replicasetStore interface {
	List() (replicasets []v1beta1.ReplicaSet, err error)
}


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
	
	client = kubeClient.ExtensionsV1beta1().RESTClient()
	glog.Infof("collect deployment with %s", client.APIVersion())
	dlw := cache.NewListWatchFromClient(client, "deployments", namespace, fields.Everything())
	dinf := cache.NewSharedInformer(dlw, &v1beta1.Deployment{}, resyncPeriod)
	dplLister := DeploymentLister(func() (deployments []v1beta1.Deployment, err error) {
		for _, c := range dinf.GetStore().List() {
			deployments = append(deployments, *(c.(*v1beta1.Deployment)))
		}
		return deployments, nil
	})
	
	glog.Infof("collect replicaset with %s", client.APIVersion())
	rslw := cache.NewListWatchFromClient(client, "replicasets", namespace, fields.Everything())
	rsinf := cache.NewSharedInformer(rslw, &v1beta1.ReplicaSet{}, resyncPeriod)
	replicaSetLister := ReplicaSetLister(func() (replicasets []v1beta1.ReplicaSet, err error) {
		for _, c := range rsinf.GetStore().List() {
			replicasets = append(replicasets, *(c.(*v1beta1.ReplicaSet)))
		}
		return replicasets, nil
	})

	prometheus.Register(newServiceCollector(podLister, dplLister, replicaSetLister))
	go pinf.Run(context.Background().Done())
	go dinf.Run(context.Background().Done())
	go rsinf.Run(context.Background().Done())
	
	//just test for informer handlers
	dinf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			glog.V(5).Infof("catch AddFunc %v", o)
		},
		DeleteFunc: func(o interface{}) {
			glog.V(5).Infof("catch DeleteFunc %v", o)
		},
		UpdateFunc: func(_, o interface{}) {
			glog.V(5).Infof("catch UpdateFunc %v", o)
		},
	})
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

func (s *ServiceCollector)displayPod(pod v1.Pod){
	glog.V(3).Infof("*****************************")
	glog.V(5).Infof("[POD]%v", pod)
	glog.V(3).Infof("Pod[%s] || %s", pod.Name, pod.Namespace)
	glog.V(3).Infof("Node: %s", pod.Spec.NodeName)
	glog.V(3).Infof("Phase: %s", pod.Status.Phase)
	glog.V(3).Infof("PodIP: %s", pod.Status.PodIP)
	glog.V(3).Infof("QOSClass: %v", pod.Status.QOSClass)
	glog.V(3).Infof("*****************************")
}

func (s *ServiceCollector)displayDeployment(dl v1beta1.Deployment){
	glog.V(3).Infof("*****************************")
	glog.V(5).Infof("[DEPLOYMENT]%v", dl)
	glog.V(3).Infof("Deployment[%s] || %s", dl.Name, dl.Namespace)
	glog.V(3).Infof("Replicas: %d", *dl.Spec.Replicas)
	glog.V(3).Infof("ReadyReplicas: %d", dl.Status.ReadyReplicas)
	glog.V(3).Infof("AvailableReplicas: %d", dl.Status.AvailableReplicas)
	glog.V(3).Infof("UnavailableReplicas: %d", dl.Status.UnavailableReplicas)
	glog.V(5).Infof("PodTemplate: %#v", dl.Spec.Template)
	glog.V(3).Infof("*****************************")
}

func (s *ServiceCollector)displayReplicaSet(rs v1beta1.ReplicaSet){
	glog.V(3).Infof("*****************************")
	glog.V(5).Infof("[ReplicaSet]%v", rs)
	glog.V(3).Infof("ReplicaSet[%s] || %s", rs.Name, rs.Namespace)
	glog.V(3).Infof("Replicas: %d", rs.Status.Replicas)
	glog.V(3).Infof("ReadyReplicas: %d", rs.Status.ReadyReplicas)
	glog.V(3).Infof("AvailableReplicas: %d", rs.Status.AvailableReplicas)
	glog.V(5).Infof("Conditions: %#v", rs.Status.Conditions)
	glog.V(3).Infof("*****************************")
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
	fmt.Printf("Collect at %v\n", time.Now())
	/*var status float64
	status = 1
	s.Status.WithLabelValues("service-a", "node1234").Set(status)
	status = 2
	s.Status.WithLabelValues("service-b", "node5678").Set(status)*/
	
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