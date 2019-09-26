package noderemove

import (
	"context"
	"fmt"
	"github.com/rancher/rancher/pkg/ref"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	dnrTaintKey = "pkds.it/do-not-resuscitate"
	clusterAutoScalerTaintKey = "ToBeDeletedByClusterAutoscaler"
)

type nodeRemove struct {
	nodePoolController	v3.NodePoolController
	nodePoolLister 		v3.NodePoolLister
	nodePools			v3.NodePoolInterface
}

func Register(ctx context.Context, management *config.ManagementContext) {
	n := &nodeRemove{
		nodePoolController: management.Management.NodePools("").Controller(),
		nodePoolLister: management.Management.NodePools("").Controller().Lister(),
		nodePools: management.Management.NodePools(""),
	}

	management.Management.Nodes("").AddLifecycle(ctx, "nodepool-noderemove", n)
}

func (n *nodeRemove) Create(obj *v3.Node) (runtime.Object, error) {
	return obj, nil
}

func (n *nodeRemove) Remove(obj *v3.Node) (runtime.Object, error) {
	//if obj == nil {
	//	return nil, nil
	//}
	//
	//for _, t := range obj.Spec.InternalNodeSpec.Taints {
	//	if t.Key == dnrTaintKey || t.Key == clusterAutoScalerTaintKey {
	//		return n.scaleDownNodePool(obj, t)
	//	}
	//}

	return obj, nil
}

func (n *nodeRemove) scaleDownNodePool(obj *v3.Node, taint v1.Taint) (runtime.Object, error) {
	namespace, name := ref.Parse(obj.Spec.NodePoolName)

	if namespace == "" || name == "" {
		return nil, fmt.Errorf("unable to determine node pool of node %s", obj.Status.NodeName)
	}

	np, err := n.nodePoolLister.Get(namespace, name)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Will resize node pool %s due to removal of node %s with taint %s=%s:%s",
		np.Spec.DisplayName, taint.Key, taint.Value, taint.Effect, obj.Status.NodeName)

	updated := np.DeepCopy()
	updated.Spec.Quantity--

	_, err = n.nodePools.Update(updated)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (n *nodeRemove) Updated(obj *v3.Node) (runtime.Object, error) {
	return obj, nil
}
