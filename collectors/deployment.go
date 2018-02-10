package collectors

import (
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

type DeploymentLister func() ([]v1beta1.Deployment, error)
func (l DeploymentLister) List() ([]v1beta1.Deployment, error) {
	return l()
}
type deploymentStore interface {
	List() (deployments []v1beta1.Deployment, err error)
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