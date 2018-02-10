package collectors

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

type PodLister func() ([]v1.Pod, error)
func (l PodLister) List() ([]v1.Pod, error) {
	return l()
}
type podStore interface {
	List() (pods []v1.Pod, err error)
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