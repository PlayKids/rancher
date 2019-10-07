package noderemove

import (
	"context"

	"github.com/rancher/rancher/pkg/ref"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	PleaseKillMeAnnotation = "nodes.pkds.it/please-kill-me"
)

func Register(ctx context.Context, management *config.ManagementContext) {
	nprc := &nodePoolRemoveController{
		nodePoolController:	management.Management.NodePools("").Controller(),
		nodePools: 			management.Management.NodePools(""),
		nodeLister:			management.Management.Nodes("").Controller().Lister(),
	}

	management.Management.NodePools("").AddLifecycle(ctx, "nodepool-noderemove", nprc)
}

// NodePool Lifecycle

type nodePoolRemoveController struct {
	nodePoolController	v3.NodePoolController
	nodePools			v3.NodePoolInterface
	nodeLister			v3.NodeLister
}

func (n *nodePoolRemoveController) Create(obj *v3.NodePool) (runtime.Object, error) {
	return obj, nil
}

func (n *nodePoolRemoveController) Remove(obj *v3.NodePool) (runtime.Object, error) {
	return obj, nil
}

func (n *nodePoolRemoveController) Updated(obj *v3.NodePool) (runtime.Object, error) {
	nodes, err := n.listNodes(obj)
	if err != nil {
		return nil, err
	}

	nodesToRemove := make([]*v3.Node, 0)

	for _, node := range nodes {
		if HasRemovalAnnotation(node) {
			nodesToRemove = append(nodesToRemove, node)
		}
	}

	if len(nodesToRemove) == 0 {
		return obj, nil
	}

	updated, err := n.nodePools.GetNamespaced(obj.Namespace, obj.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if updated.Spec.Quantity != len(nodes) {
		logrus.Infof("Another update is in progress for %s, will skip for now", updated.Spec.HostnamePrefix)
		n.nodePoolController.Enqueue(updated.Namespace, updated.Namespace)

		return updated, nil
	}

	logrus.Infof("Found %d nodes to remove from %s", len(nodesToRemove), obj.Spec.HostnamePrefix)

	desiredQuantity := len(nodes) - len(nodesToRemove)
	logrus.Infof("Changing node quantity of %s from %d to %d",
		obj.Spec.HostnamePrefix, updated.Spec.Quantity, desiredQuantity)

	updated.Spec.Quantity = desiredQuantity

	return updated, nil
}

func (n *nodePoolRemoveController) listNodes(nodePool *v3.NodePool) ([]*v3.Node, error) {
	allNodes, err := n.nodeLister.List(nodePool.Namespace, labels.Everything())
	if err != nil {
		return nil, err
	}

	var nodes []*v3.Node
	for _, node := range allNodes {
		_, nodePoolName := ref.Parse(node.Spec.NodePoolName)

		if nodePoolName == nodePool.Name {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

func HasRemovalAnnotation(node *v3.Node) bool {
	return node.ObjectMeta.Annotations[PleaseKillMeAnnotation] == "true"
}
