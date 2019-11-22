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

package main

import (
	"flag"
	"html/template"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/aws"
	klog "k8s.io/klog/v2"
	"os"
)

var packageTemplate = template.Must(template.New("").Parse(`/*
Copyright The Kubernetes Authors.

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

// This file was generated by go generate; DO NOT EDIT

package aws

type InstanceType struct {
	InstanceType string
	VCPU         int64
	MemoryMb     int64
	GPU          int64
	Storage      int64
}

// InstanceTypes is a map of ec2 resources
var InstanceTypes = map[string]*InstanceType{
{{- range .InstanceTypes }}
	"{{ .InstanceType }}": {
		InstanceType: "{{ .InstanceType }}",
		VCPU:         {{ .VCPU }},
		MemoryMb:     {{ .MemoryMb }},
		GPU:          {{ .GPU }},
		Storage:      {{ .Storage }},
	},
{{- end }}
}
`))

func main() {
	var region = flag.String("region", "", "aws region you'd like to generate instances from."+
		"It will populate list from all regions if region is not specified.")
	flag.Parse()
	defer klog.Flush()

	instanceTypes, err := aws.GenerateEC2InstanceTypes(*region)
	if err != nil {
		klog.Fatal(err)
	}

	f, err := os.Create("ec2_instance_types.go")
	if err != nil {
		klog.Fatal(err)
	}

	defer f.Close()

	err = packageTemplate.Execute(f, struct {
		InstanceTypes map[string]*aws.InstanceType
	}{
		InstanceTypes: instanceTypes,
	})

	if err != nil {
		klog.Fatal(err)
	}
}
