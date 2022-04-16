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

	// Create cluster
	if err := node.Command(
		"helm",
		"install",
		"--repo",
		"https://prcloud10.github.io/kind/cluster/",
		"cluster1",
		"cluster",
		"--set", "ip="+a.Ip,
		"--set", "workers="+a.Workers,
		"--set", "controllers="+a.Controllers,
		"--set", "k8version="+a.K8version,
	).Run(); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	// mark success
	ctx.Status.End(true)
	return nil
}
