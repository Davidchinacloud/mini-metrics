package collectors

import (
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

type ReplicaSetLister func() ([]v1beta1.ReplicaSet, error)
func (l ReplicaSetLister) List() ([]v1beta1.ReplicaSet, error) {
	return l()
}
type replicasetStore interface {
	List() (replicasets []v1beta1.ReplicaSet, err error)
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