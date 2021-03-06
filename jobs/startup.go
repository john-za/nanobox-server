// Copyright (c) 2014 Pagoda Box Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public License,
// v. 2.0. If a copy of the MPL was not distributed with this file, You can
// obtain one at http://mozilla.org/MPL/2.0/.

package jobs

//
import (
	"github.com/nanobox-io/nanobox-server/config"
	"github.com/nanobox-io/nanobox-server/util/docker"
	"github.com/nanobox-io/nanobox-server/util/worker"
)

//
type Startup struct{}

// process on startup
func (j *Startup) Process() {
	config.Log.Info("starting startup job")

	docker.RemoveContainer("exec1")
	box := CombinedBoxfile(false)

	configureRoutes(*box)
	configurePorts(*box)

	// we also need to set up a ssh tunnel for each running docker container
	// this is easiest to do by creating a ServiceEnv job and working it
	worker := worker.New()
	worker.Blocking = true
	worker.Concurrent = true

	serviceContainers, _ := docker.ListContainers("service")
	for _, container := range serviceContainers {
		s := ServiceEnv{UID: container.Config.Labels["uid"], FirstTime: true}
		worker.Queue(&s)
	}

	worker.Process()
}
