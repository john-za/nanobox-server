// Copyright (c) 2014 Pagoda Box Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public License,
// v. 2.0. If a copy of the MPL was not distributed with this file, You can
// obtain one at http://mozilla.org/MPL/2.0/.

//
package jobs

import (
	"github.com/nanobox-io/nanobox-boxfile"
	"github.com/nanobox-io/nanobox-golang-stylish"
	"github.com/nanobox-io/nanobox-server/config"
	"github.com/nanobox-io/nanobox-server/util"
	"github.com/nanobox-io/nanobox-server/util/script"
)

//
type Restart struct {
	UID     string
	Success bool
	Boxfile boxfile.Boxfile
}

// Proccess syncronies your docker containers with the boxfile specification
func (j *Restart) Process() {
	// add a lock so the service wont go down whil im running
	util.Lock()
	defer util.Unlock()

	j.Success = false

	util.LogInfo(stylish.Bullet("Restarting app in %s container...", j.UID))
	box := CombinedBoxfile(false)
	// restart payload
	payload := map[string]interface{}{
		"platform":    "local",
		"boxfile":     box.Node(j.UID).Parsed,
		"logtap_host": config.LogtapHost,
		"uid":         j.UID,
	}

	// run restart hook (blocking)
	if _, err := script.Exec("default-restart", j.UID, payload); err != nil {
		util.LogInfo("ERROR %v\n", err)
		return
	}

	j.Success = true
}
