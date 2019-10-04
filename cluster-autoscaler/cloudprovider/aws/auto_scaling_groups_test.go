/*
Copyright 2017 The Kubernetes Authors.

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

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildAsg(t *testing.T) {
	asgCache := &asgCache{}

	asg, err := asgCache.buildAsgFromSpec("1:5:test-asg")
	assert.NoError(t, err)
	assert.Equal(t, asg.minSize, 1)
	assert.Equal(t, asg.maxSize, 5)
	assert.Equal(t, asg.Name, "test-asg")

	_, err = asgCache.buildAsgFromSpec("a")
	assert.Error(t, err)
	_, err = asgCache.buildAsgFromSpec("a:b:c")
	assert.Error(t, err)
	_, err = asgCache.buildAsgFromSpec("1:")
	assert.Error(t, err)
	_, err = asgCache.buildAsgFromSpec("1:2:")
	assert.Error(t, err)
}

func validateAsg(t *testing.T, asg *asg, name string, minSize int, maxSize int) {
	assert.Equal(t, name, asg.Name)
	assert.Equal(t, minSize, asg.minSize)
	assert.Equal(t, maxSize, asg.maxSize)
}

func Test_asgCache_buildFilters(t *testing.T) {
	tests := []struct {
		name                  string
		asgAutoDiscoverySpecs []cloudprovider.ASGAutoDiscoveryConfig
		want                  []*autoscaling.Filter
	}{
		{
			name:                  "Standard single tag filter",
			asgAutoDiscoverySpecs: []cloudprovider.ASGAutoDiscoveryConfig{{Tags: map[string]string{"k8s.io/cluster-autoscaler/enabled": ""}}},
			want: []*autoscaling.Filter{{
				Name:   aws.String("key"),
				Values: []*string{aws.String("k8s.io/cluster-autoscaler/enabled")},
			}},
		},
		{
			name: "Standard multiple tag filter",
			asgAutoDiscoverySpecs: []cloudprovider.ASGAutoDiscoveryConfig{{
				Tags: map[string]string{
					"node-role.kubernetes.io/cluster-autoscaler": "",
					"k8s.io/cluster-autoscaler/enabled":          "",
				}}},
			want: []*autoscaling.Filter{
				{
					Name:   aws.String("key"),
					Values: []*string{aws.String("node-role.kubernetes.io/cluster-autoscaler")},
				},
				{
					Name:   aws.String("key"),
					Values: []*string{aws.String("k8s.io/cluster-autoscaler/enabled")},
				},
			},
		},
		{
			name: "Standard multiple tag filter with values",
			asgAutoDiscoverySpecs: []cloudprovider.ASGAutoDiscoveryConfig{{
				Tags: map[string]string{
					"node-role.kubernetes.io/cluster-autoscaler": "",
					"k8s.io/cluster-autoscaler/enabled":          "",
					"beta.kubernetes.io/instance-type":           "c5.xlarge",
				}}},
			want: []*autoscaling.Filter{
				{
					Name:   aws.String("key"),
					Values: []*string{aws.String("node-role.kubernetes.io/cluster-autoscaler")},
				},
				{
					Name:   aws.String("key"),
					Values: []*string{aws.String("k8s.io/cluster-autoscaler/enabled")},
				},
				{
					Name:   aws.String("key"),
					Values: []*string{aws.String("beta.kubernetes.io/instance-type")},
				},
				{
					Name:   aws.String("value"),
					Values: []*string{aws.String("c5.xlarge")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &asgCache{
				asgAutoDiscoverySpecs: tt.asgAutoDiscoverySpecs,
			}
			got := m.buildFilters()
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_asgCache_extractAsgNames(t *testing.T) {
	tests := []struct {
		name                  string
		asgAutoDiscoverySpecs []cloudprovider.ASGAutoDiscoveryConfig
		tags                  []*autoscaling.TagDescription
		want                  []string
	}{
		{
			name: "", asgAutoDiscoverySpecs: []cloudprovider.ASGAutoDiscoveryConfig{{
				Tags: map[string]string{
					"node-role.kubernetes.io/cluster-autoscaler": "",
					"k8s.io/cluster-autoscaler/enabled":          "",
					"beta.kubernetes.io/instance-type":           "c5.xlarge",
				}}, {
				Tags: map[string]string{
					"node-role.kubernetes.io/cluster-autoscaler": "",
					"k8s.io/cluster-autoscaler/enabled":          "",
				}},
			},
			tags: []*autoscaling.TagDescription{
				{
					Key:               aws.String("node-role.kubernetes.io/cluster-autoscaler"),
					PropagateAtLaunch: nil,
					ResourceId:        aws.String("asg-0"),
					ResourceType:      aws.String("auto-scaling-group"),
					Value:             aws.String(""),
				}, {
					Key:               aws.String("k8s.io/cluster-autoscaler/enabled"),
					PropagateAtLaunch: nil,
					ResourceId:        aws.String("asg-0"),
					ResourceType:      aws.String("auto-scaling-group"),
					Value:             aws.String(""),
				}, {
					Key:               aws.String("k8s.io/cluster-autoscaler/enabled"),
					PropagateAtLaunch: nil,
					ResourceId:        aws.String("asg-1"),
					ResourceType:      aws.String("auto-scaling-group"),
					Value:             aws.String(""),
				}, {
					Key:               aws.String("beta.kubernetes.io/instance-type"),
					PropagateAtLaunch: nil,
					ResourceId:        aws.String("asg-1"),
					ResourceType:      aws.String("auto-scaling-group"),
					Value:             aws.String(""),
				}, {
					Key:               aws.String("node-role.kubernetes.io/cluster-autoscaler"),
					PropagateAtLaunch: nil,
					ResourceId:        aws.String("asg-1"),
					ResourceType:      aws.String("auto-scaling-group"),
					Value:             aws.String(""),
				}, {
					Key:               aws.String("k8s.io/cluster-autoscaler/enabled"),
					PropagateAtLaunch: nil,
					ResourceId:        aws.String("asg-2"),
					ResourceType:      aws.String("auto-scaling-group"),
					Value:             aws.String(""),
				},
			},
			want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &asgCache{
				asgAutoDiscoverySpecs: tt.asgAutoDiscoverySpecs,
			}
			got := m.extractAsgNames(tt.tags)
			assert.Equal(t, got, tt.want)
		})
	}
}
