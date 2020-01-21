package taints

import (
	"strings"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/utils"
)

const nidhoggTaintPrefix = "nidhogg.uswitch.com"

func hasNidhoggTaint(node *apiv1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if strings.HasPrefix(taint.Key, nidhoggTaintPrefix) {
			return true
		}
	}
	return false
}

func FilterOutNodesWithTaints(allNodes, readyNodes []*apiv1.Node) ([]*apiv1.Node, []*apiv1.Node) {
	var newAllNodes []*apiv1.Node
	var newReadyNodes []*apiv1.Node
	nodesWithTaints := make(map[string]*apiv1.Node)

	for _, node := range readyNodes {
		if hasNidhoggTaint(node) {
			nodesWithTaints[node.Name] = utils.GetUnreadyNodeCopy(node)
		} else {
			newReadyNodes = append(newReadyNodes, node)
		}
	}

	for _, node := range allNodes {
		if newNode, found := nodesWithTaints[node.Name]; found {
			newAllNodes = append(newAllNodes, newNode)
		} else {
			newAllNodes = append(newAllNodes, node)
		}
	}

	return newAllNodes, newReadyNodes
}
