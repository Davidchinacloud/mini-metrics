package collectors

import (
	"time"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
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

type StatusInfo struct {
	name string
	namespace string
	status int
}

type ServiceCollector struct {
	fastSerivceStatus	[]*prometheus.GaugeVec
	pStore				podStore
	dStore				deploymentStore
	rStore      		replicasetStore
	statues             chan StatusInfo
	done                chan struct{}
	mu                  sync.Mutex
}

func RegisterServiceCollector(kubeClient kubernetes.Interface, namespace string, ch chan struct{}) {
	podLister := registerPodCollector(kubeClient, namespace)
	dplLister := registerDeploymentCollector(kubeClient, namespace)
	replicaSetLister := registerReplicaSetCollector(kubeClient, namespace)
	sc := newServiceCollector(podLister, dplLister, replicaSetLister, ch)
	prometheus.Register(sc)
	
	// just test k8s-client
	testNodeListUpdate(kubeClient)
	
	//TODO: need close goroutine such as signalKillHandle..
	go sc.waitStatus()	
}

func newServiceCollector(ps podStore, ds deploymentStore, rs replicasetStore, ch chan struct{})*ServiceCollector{
	labels := make(prometheus.Labels)

	return &ServiceCollector{
		fastSerivceStatus: []*prometheus.GaugeVec{ 
		prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_building",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),
		prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_failed",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),
		prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_runnning",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),
		prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "mock",
				Name:        "service_status_stopped",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service_name", "namespace"},
		),
		},	
		pStore: ps,
		dStore: ds,
		rStore: rs,
		statues: make(chan StatusInfo),
		done:   ch,
	}
}

func (s *ServiceCollector)calculateStatus(rs v1beta1.ReplicaSet, pods *[]v1.Pod){
	var sinfo = StatusInfo{
		name: rs.Name,
		namespace: rs.Namespace,
	}
	
	//Record Deployments/DaemonSets name as service name
	owners := rs.GetOwnerReferences()
	if len(owners) > 0 {
		if owners[0].Controller != nil {
			sinfo.name = owners[0].Name
		}
	}
	
	waitingReason := func(cs v1.ContainerStatus)string{
		if cs.State.Waiting == nil {
			return ""
		}
		return cs.State.Waiting.Reason
	}
	
	//TODO: move pod-ref-rs out
	hasErrorPod := func(rs v1beta1.ReplicaSet, pods *[]v1.Pod)bool{
		for _, pod := range *pods{
			ow := pod.GetOwnerReferences()
			if len(ow) > 0 {
				if ow[0].Name == rs.Name {
					for _, cs := range pod.Status.ContainerStatuses {
						reason := waitingReason(cs)
						if reason == "ImagePullBackOff" || reason == "ErrImagePull"  {
							return true
						}
					}
				}
			}
		}
		return false
	}
	  
	if rs.Status.AvailableReplicas == rs.Status.Replicas &&
	rs.Status.AvailableReplicas > 0{
		sinfo.status = statusRunning
	} else if rs.Status.ReadyReplicas < *rs.Spec.Replicas {
		if hasErrorPod(rs, pods) {
			sinfo.status = statusFailed
		} else {
			sinfo.status = statusBuilding
		}
	} else if *rs.Spec.Replicas == 0 {
		sinfo.status = statusStopped
	} else {
		sinfo.status = statusBuilding
	}
	s.statues<-sinfo
}

func (s *ServiceCollector)calculateSetValue(){
	replicasets, err := s.rStore.List()
	if err != nil {
		glog.Errorf("listing replicasets failed: %s", err)
	} 
	pods, err := s.pStore.List()
	if err != nil {
		glog.Errorf("listing pods failed: %s", err)
	} 
	for _, r := range replicasets {
		s.calculateStatus(r, &pods)
	}
}

func (s *ServiceCollector)waitStatus(){
	for {
		select {
			case recv := <-s.statues:
				s.mu.Lock()
				for k, status := range s.fastSerivceStatus {
					if k == recv.status {
						status.WithLabelValues(recv.name, recv.namespace).Set(1)
					} else {
						status.WithLabelValues(recv.name, recv.namespace).Set(0)
					}
				}
				s.mu.Unlock()
			case <-s.done:
				glog.V(3).Infof("Received SIGTERM, exiting gracefully..")
				return	
		}
	}
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

	s.mu.Lock()
	for _, metric := range s.collectorList() {
		metric.Collect(ch)
	}
	s.mu.Unlock()
}

func (s *ServiceCollector) collectorList() []prometheus.Collector {
	var cl []prometheus.Collector
	for _, metrics := range s.fastSerivceStatus {
		cl = append(cl, metrics)
	}
	return cl
}

func boolFloat64(b bool)float64{
	if b {
		return 1
	}
	return 0
}

// just test for k8s-client once list and update
func testNodeListUpdate(kubeClient kubernetes.Interface){
	nodes, err := kubeClient.Core().Nodes().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("List nodes failed: %v", err)
		return
	}
	if len(nodes.Items) > 0 {
		node := nodes.Items[0]
		glog.V(3).Infof("Nodes: %#v", node)
		node.Annotations["checked"] = "true"
		_, err = kubeClient.Core().Nodes().Update(&node)
		if err != nil {
			glog.Errorf("Update node failed: %v", err)
		}
	}
}