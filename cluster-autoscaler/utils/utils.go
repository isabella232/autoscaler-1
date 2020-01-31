/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
)

// GetNodeGroupSizeMap return a map of node group id and its target size
func GetNodeGroupSizeMap(cloudProvider cloudprovider.CloudProvider) map[string]int {
	nodeGroupSize := make(map[string]int)
	for _, nodeGroup := range cloudProvider.NodeGroups() {
		size, err := nodeGroup.TargetSize()
		if err != nil {
			klog.Errorf("Error while checking node group size %s: %v", nodeGroup.Id(), err)
			continue
		}
		nodeGroupSize[nodeGroup.Id()] = size
	}
	return nodeGroupSize
}

// FilterOutNodes filters out nodesToFilterOut from nodes
func FilterOutNodes(nodes []*apiv1.Node, nodesToFilterOut []*apiv1.Node) []*apiv1.Node {
	var filtered []*apiv1.Node
	for _, node := range nodes {
		found := false
		for _, nodeToFilter := range nodesToFilterOut {
			if nodeToFilter.Name == node.Name {
				found = true
			}
		}
		if !found {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// GetUnreadyNodeCopy returns a deepcopy of the node with its status overridden to "NotReady"
func GetUnreadyNodeCopy(node *apiv1.Node) *apiv1.Node {
	newNode := node.DeepCopy()
	newReadyCondition := apiv1.NodeCondition{
		Type:               apiv1.NodeReady,
		Status:             apiv1.ConditionFalse,
		LastTransitionTime: node.CreationTimestamp,
	}
	newNodeConditions := []apiv1.NodeCondition{newReadyCondition}
	for _, condition := range newNode.Status.Conditions {
		if condition.Type != apiv1.NodeReady {
			newNodeConditions = append(newNodeConditions, condition)
		}
	}
	newNode.Status.Conditions = newNodeConditions
	return newNode
}
