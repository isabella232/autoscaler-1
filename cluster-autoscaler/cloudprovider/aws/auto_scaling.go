/*
Copyright 2016 The Kubernetes Authors.

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
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"k8s.io/autoscaler/cluster-autoscaler/metrics"
	"k8s.io/klog"
	"sync"
	"time"
)

// autoScaling is the interface represents a specific aspect of the auto-scaling service provided by AWS SDK for use in CA
type autoScaling interface {
	DescribeAutoScalingGroupsPages(input *autoscaling.DescribeAutoScalingGroupsInput, fn func(*autoscaling.DescribeAutoScalingGroupsOutput, bool) bool) error
	DescribeLaunchConfigurations(*autoscaling.DescribeLaunchConfigurationsInput) (*autoscaling.DescribeLaunchConfigurationsOutput, error)
	DescribeTagsPages(input *autoscaling.DescribeTagsInput, fn func(*autoscaling.DescribeTagsOutput, bool) bool) error
	SetDesiredCapacity(input *autoscaling.SetDesiredCapacityInput) (*autoscaling.SetDesiredCapacityOutput, error)
	TerminateInstanceInAutoScalingGroup(input *autoscaling.TerminateInstanceInAutoScalingGroupInput) (*autoscaling.TerminateInstanceInAutoScalingGroupOutput, error)
}

// autoScalingWrapper provides several utility methods over the auto-scaling service provided by AWS SDK
type autoScalingWrapper struct {
	autoScaling
	lcMu                                 sync.Mutex
	launchConfigurationInstanceTypeCache map[string]string
}

func (m autoScalingWrapper) populateLaunchConfigurationInstanceTypeCache(autoscalingGroups []*autoscaling.Group) error {
	m.lcMu.Lock()
	defer m.lcMu.Unlock()
	launchToInstanceType := make(map[string]string)

	var launchConfigToQuery []*string

	for _, asg := range autoscalingGroups {
		if asg == nil {
			continue
		}
		if asg.LaunchConfigurationName == nil {
			continue
		}
		i, ok := m.launchConfigurationInstanceTypeCache[*asg.LaunchConfigurationName]
		if ok {
			launchToInstanceType[*asg.LaunchConfigurationName] = i
			continue
		}
		launchConfigToQuery = append(launchConfigToQuery, asg.LaunchConfigurationName)
	}
	if len(launchConfigToQuery) == 0 {
		klog.V(4).Infof("%d launch configurations already in cache", len(autoscalingGroups))
		return nil
	}
	klog.V(4).Infof("%d launch configurations to query", len(launchConfigToQuery))

	alreadyRetry := false
	for i := 0; i < len(launchConfigToQuery); i += 50 {
		end := i + 50

		if end > len(launchConfigToQuery) {
			end = len(launchConfigToQuery)
		}
		for {
			params := &autoscaling.DescribeLaunchConfigurationsInput{
				LaunchConfigurationNames: launchConfigToQuery[i:end],
				MaxRecords:               aws.Int64(50),
			}
			start := time.Now()
			r, err := m.DescribeLaunchConfigurations(params)
			metrics.ObserveCloudProviderQuery("aws", "DescribeLaunchConfigurations", err, start)
			if err == nil {
				for _, lc := range r.LaunchConfigurations {
					launchToInstanceType[*lc.LaunchConfigurationName] = *lc.InstanceType
					m.launchConfigurationInstanceTypeCache[*lc.LaunchConfigurationName] = *lc.InstanceType
				}
				break
			}
			if !alreadyRetry && request.IsErrorThrottle(err) {
				alreadyRetry = true
				klog.Warningf("DescribeLaunchConfigurations retry: %v", err)
				continue
			}
			return err
		}
	}

	klog.V(4).Infof("Successfully updated %d launch configurations, replacing current launch configuration cache from %d to %d", len(launchConfigToQuery), len(m.launchConfigurationInstanceTypeCache), len(launchToInstanceType))
	m.launchConfigurationInstanceTypeCache = launchToInstanceType
	return nil
}

func (m autoScalingWrapper) getInstanceTypeByLCName(name string) (string, error) {
	m.lcMu.Lock()
	defer m.lcMu.Unlock()

	if instanceType, found := m.launchConfigurationInstanceTypeCache[name]; found {
		return instanceType, nil
	}

	params := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{aws.String(name)},
		MaxRecords:               aws.Int64(1),
	}
	start := time.Now()
	launchConfigurations, err := m.DescribeLaunchConfigurations(params)
	metrics.ObserveCloudProviderQuery("aws", "DescribeLaunchConfigurations", err, start)
	if err != nil {
		klog.V(4).Infof("Failed LaunchConfiguration info request for %s: %v", name, err)
		return "", err
	}
	if len(launchConfigurations.LaunchConfigurations) < 1 {
		return "", fmt.Errorf("unable to get first LaunchConfiguration for %s", name)
	}
	instanceType := *launchConfigurations.LaunchConfigurations[0].InstanceType
	m.launchConfigurationInstanceTypeCache[name] = instanceType
	return instanceType, nil
}

func (m *autoScalingWrapper) getAutoscalingGroupsByNames(names []string) ([]*autoscaling.Group, error) {
	if len(names) == 0 {
		return nil, nil
	}

	var asgs []*autoscaling.Group
	alreadyRetry := false

	// AWS only accepts up to 50 ASG names as input, describe them in batches
	for i := 0; i < len(names); i += maxAsgNamesPerDescribe {
		end := i + maxAsgNamesPerDescribe

		if end > len(names) {
			end = len(names)
		}

		for {
			input := &autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: aws.StringSlice(names[i:end]),
				MaxRecords:            aws.Int64(maxAsgNamesPerDescribe),
			}
			start := time.Now()
			err := m.DescribeAutoScalingGroupsPages(input, func(output *autoscaling.DescribeAutoScalingGroupsOutput, _ bool) bool {
				asgs = append(asgs, output.AutoScalingGroups...)
				// We return true while we want to be called with the next page of
				// results, if any.
				return true
			})
			metrics.ObserveCloudProviderQuery("aws", "DescribeAutoScalingGroupsPages", err, start)
			if err == nil {
				break
			}
			if !alreadyRetry && request.IsErrorThrottle(err) {
				alreadyRetry = true
				klog.Warningf("DescribeAutoScalingGroupsPages retry: %v", err)
				continue
			}
			return nil, err
		}
	}

	return asgs, nil
}

func (m *autoScalingWrapper) getAutoscalingGroupNamesByTags(kvs map[string]string) ([]string, error) {
	// DescribeTags does an OR query when multiple filters on different tags are
	// specified. In other words, DescribeTags returns [asg1, asg1] for keys
	// [t1, t2] when there's only one asg tagged both t1 and t2.
	filters := []*autoscaling.Filter{}
	for key, value := range kvs {
		filter := &autoscaling.Filter{
			Name:   aws.String("key"),
			Values: []*string{aws.String(key)},
		}
		filters = append(filters, filter)
		if value != "" {
			filters = append(filters, &autoscaling.Filter{
				Name:   aws.String("value"),
				Values: []*string{aws.String(value)},
			})
		}
	}

	tags := []*autoscaling.TagDescription{}
	input := &autoscaling.DescribeTagsInput{
		Filters:    filters,
		MaxRecords: aws.Int64(maxRecordsReturnedByAPI),
	}
	start := time.Now()
	err := m.DescribeTagsPages(input, func(out *autoscaling.DescribeTagsOutput, _ bool) bool {
		tags = append(tags, out.Tags...)
		// We return true while we want to be called with the next page of
		// results, if any.
		return true
	})
	metrics.ObserveCloudProviderQuery("aws", "DescribeTagsPages", err, start)
	if err != nil {
		return nil, err
	}

	// According to how DescribeTags API works, the result contains ASGs which
	// not all but only subset of tags are associated. Explicitly select ASGs to
	// which all the tags are associated so that we won't end up calling
	// DescribeAutoScalingGroups API multiple times on an ASG.
	asgNames := []string{}
	asgNameOccurrences := make(map[string]int)
	for _, t := range tags {
		asgName := aws.StringValue(t.ResourceId)
		occurrences := asgNameOccurrences[asgName] + 1
		if occurrences >= len(kvs) {
			asgNames = append(asgNames, asgName)
		}
		asgNameOccurrences[asgName] = occurrences
	}

	return asgNames, nil
}
