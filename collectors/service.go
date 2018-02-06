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
	resyncPeriod = 15 * time.Second
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

type ServiceCollector struct {
	Status		*prometheus.GaugeVec
	pStore		podStore
	dStore		deploymentStore
}


func RegisterServiceCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface, 
	namespace string) {
	//registry.MustRegister()
	fmt.Printf("just log for use var registry %v\n", registry)
	
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

	prometheus.Register(newServiceCollector(podLister, dplLister))
	go pinf.Run(context.Background().Done())
	go dinf.Run(context.Background().Done())
}


func newServiceCollector(ps podStore, ds deploymentStore)*ServiceCollector{
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
		pStore: ps,
		dStore: ds,
	}
}

func (s *ServiceCollector) collectorList() []prometheus.Collector {
	return []prometheus.Collector{
		s.Status,
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
	glog.V(3).Infof("Replicas: %s", dl.Spec.Replicas)
	glog.V(3).Infof("ReadyReplicas: %d", dl.Status.ReadyReplicas)
	glog.V(3).Infof("AvailableReplicas: %d", dl.Status.AvailableReplicas)
	glog.V(3).Infof("UnavailableReplicas: %d", dl.Status.UnavailableReplicas)
	glog.V(3).Infof("*****************************")
}

func (s *ServiceCollector)collect()error{
	fmt.Printf("Collect at %v\n", time.Now())
	var status float64
	status = 1
	s.Status.WithLabelValues("node1234").Set(status)
	status = 2
	s.Status.WithLabelValues("node5678").Set(status)
	
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

