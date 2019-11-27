package readiness

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/utils"
)

const nidhoggReadiness = "nidhogg.uswitch.com/first-time-ready"

func FilterOutNodesWithLabels(allNodes, readyNodes []*apiv1.Node) ([]*apiv1.Node, []*apiv1.Node) {
	var newAllNodes []*apiv1.Node
	var newReadyNodes []*apiv1.Node
	nodesWithoutLabel := make(map[string]*apiv1.Node)

	for _, node := range readyNodes {
		_, ok := node.Annotations[nidhoggReadiness]
		if ok {
			newReadyNodes = append(newReadyNodes, node)
			continue
		}
		nodesWithoutLabel[node.Name] = utils.GetUnreadyNodeCopy(node)
	}

	for _, node := range allNodes {
		if newNode, found := nodesWithoutLabel[node.Name]; found {
			newAllNodes = append(newAllNodes, newNode)
		} else {
			newAllNodes = append(newAllNodes, node)
		}
	}

	return newAllNodes, newReadyNodes
}
