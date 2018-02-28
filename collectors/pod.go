package collectors

import (
	"strconv"
	
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/fields"
)

type PodLister func() ([]v1.Pod, error)
func (l PodLister) List() ([]v1.Pod, error) {
	return l()
}
type podStore interface {
	List() (pods []v1.Pod, err error)
}

func registerPodCollector(kubeClient kubernetes.Interface, namespace string)PodLister{
	client := kubeClient.CoreV1().RESTClient()
	glog.Infof("collect pod with %s", client.APIVersion())
	plw := cache.NewListWatchFromClient(client, "pods", namespace, fields.Everything())
	pinf := cache.NewSharedInformer(plw, &v1.Pod{}, resyncPeriod)
	go pinf.Run(context.Background().Done())
	
	podLister := PodLister(func() (pods []v1.Pod, err error) {
		for _, m := range pinf.GetStore().List() {
			pods = append(pods, *m.(*v1.Pod))
		}
		return pods, nil
	})
	return podLister
}

func (s *ServiceCollector)displayPod(pod v1.Pod){
	waitingReason := func(cs v1.ContainerStatus)string{
		if cs.State.Waiting == nil {
			return ""
		}
		return cs.State.Waiting.Reason
	}
	owners := pod.GetOwnerReferences()
	
	glog.V(3).Infof("*****************************")
	glog.V(5).Infof("[POD]%v", pod)
	glog.V(3).Infof("Pod[%s] || %s", pod.Name, pod.Namespace)
	glog.V(3).Infof("Node: %s", pod.Spec.NodeName)
	glog.V(3).Infof("Phase: %s", pod.Status.Phase)
	for _, cs := range pod.Status.ContainerStatuses {
		glog.V(3).Infof("Reason: %s", waitingReason(cs))
	}
	if len(owners) > 0 {
		for _, owner := range owners{
			if owner.Controller != nil {
				glog.V(3).Infof("Owner: %s, %s, %s", 
					owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller))
			} else {
				glog.V(3).Infof("Owner: %s, %s, %s", 
					owner.Kind, owner.Name, "false")
			}
		}
	}
	glog.V(3).Infof("Reason: %s", pod.Status.Reason)
	glog.V(3).Infof("PodIP: %s", pod.Status.PodIP)
	glog.V(3).Infof("QOSClass: %v", pod.Status.QOSClass)
	glog.V(3).Infof("*****************************")
}