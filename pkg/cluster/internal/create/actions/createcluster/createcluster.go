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

// Package installcapi implements the install CAPICLUSTER action
package createcluster

import (
	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
)

type action struct {
	Hostname    string
	Ip          string
	Workers     string
	Controllers string
	K8version   string
}

// NewAction returns a new action for installing default CNI
func NewAction(workers string, controllers string, hostname string, ip string, k8version string) actions.Action {
	return &action{
		Workers:     workers,
		Controllers: controllers,
		Ip:          ip,
		K8version:   k8version,
		Hostname:    hostname,
	}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Creating cluster 🔌")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	node := controlPlanes[0] // kind expects at least one always

	str := `apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
	name: byoh-cluster-md-0
	namespace: default
spec:
	template:
	spec: {}
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
	labels:
	cni: byoh-cluster-crs-0
	crs: "true"
	name: byoh-cluster
	namespace: default
spec:
	clusterNetwork:
	pods:
		cidrBlocks:
		- 192.168.0.0/16
	serviceDomain: cluster.local
	services:
		cidrBlocks:
		- 10.128.0.0/12
	controlPlaneRef:
	apiVersion: controlplane.cluster.x-k8s.io/v1beta1
	kind: KubeadmControlPlane
	name: byoh-cluster-control-plane
	infrastructureRef:
	apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
	kind: ByoCluster
	name: byoh-cluster
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
	name: byoh-cluster-md-0
	namespace: default
spec:
	clusterName: byoh-cluster
	replicas: ` + a.Workers + `
	selector:
	matchLabels: null
	template:
	metadata:
		labels:
		nodepool: pool1
	spec:
		bootstrap:
		configRef:
			apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
			kind: KubeadmConfigTemplate
			name: byoh-cluster-md-0
		clusterName: byoh-cluster
		infrastructureRef:
		apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
		kind: ByoMachineTemplate
		name: byoh-cluster-md-0
		version: ` + a.K8version + `
---
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: KubeadmControlPlane
metadata:
	labels:
	nodepool: pool0
	name: byoh-cluster-control-plane
	namespace: default
spec:
	kubeadmConfigSpec:
	clusterConfiguration:
		apiServer:
		certSANs:
		- localhost
		- 127.0.0.1
		- 0.0.0.0
		- host.docker.internal
		controllerManager:
		extraArgs:
			enable-hostpath-provisioner: "true"
	files:
	- content: |
		apiVersion: v1
		kind: Pod
		metadata:
			creationTimestamp: null
			name: kube-vip
			namespace: kube-system
		spec:
			containers:
			- args:
			- start
			env:
			- name: vip_arp
				value: "true"
			- name: vip_leaderelection
				value: "true"
			- name: vip_address
				value: ` + a.Ip + `
			- name: vip_interface
				value: {{ .DefaultNetworkInterfaceName }}
			- name: vip_leaseduration
				value: "15"
			- name: vip_renewdeadline
				value: "10"
			- name: vip_retryperiod
				value: "2"
			image: ghcr.io/kube-vip/kube-vip:v0.3.5
			imagePullPolicy: IfNotPresent
			name: kube-vip
			resources: {}
			securityContext:
				capabilities:
				add:
				- NET_ADMIN
				- SYS_TIME
			volumeMounts:
			- mountPath: /etc/kubernetes/admin.conf
				name: kubeconfig
			hostNetwork: true
			volumes:
			- hostPath:
				path: /etc/kubernetes/admin.conf
				type: FileOrCreate
			name: kubeconfig
		status: {}
		owner: root:root
		path: /etc/kubernetes/manifests/kube-vip.yaml
	initConfiguration:
		nodeRegistration:
		criSocket: /var/run/containerd/containerd.sock
		ignorePreflightErrors:
		- Swap
		- DirAvailable--etc-kubernetes-manifests
		- FileAvailable--etc-kubernetes-kubelet.conf
	joinConfiguration:
		nodeRegistration:
		criSocket: /var/run/containerd/containerd.sock
		ignorePreflightErrors:
		- Swap
		- DirAvailable--etc-kubernetes-manifests
		- FileAvailable--etc-kubernetes-kubelet.conf
	machineTemplate:
	infrastructureRef:
		apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
		kind: ByoMachineTemplate
		name: byoh-cluster-control-plane
		namespace: default
	replicas: ` + a.Controllers + `
	version: ` + a.K8version + `
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: ByoCluster
metadata:
	name: byoh-cluster
	namespace: default
spec:
	bundleLookupBaseRegistry: projects.registry.vmware.com/cluster_api_provider_bringyourownhost
	bundleLookupTag: 1.22.4
	controlPlaneEndpoint:
	host: ` + a.Ip + `
	port: 6443
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: ByoMachineTemplate
metadata:
	name: byoh-cluster-md-0
	namespace: default
spec:
	template:
	spec: {}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: ByoMachineTemplate
metadata:
	name: byoh-cluster-control-plane
	namespace: default
spec:
	template:
	spec: {}`

	// Create cluster
	if err := node.Command(
		"echo",
		str,
		"|",
		"kubectl",
		"apply",
		"-",
	).Run(); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	// mark success
	ctx.Status.End(true)
	return nil
}
