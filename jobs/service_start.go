// Copyright (c) 2014 Pagoda Box Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public License,
// v. 2.0. If a copy of the MPL was not distributed with this file, You can
// obtain one at http://mozilla.org/MPL/2.0/.

package jobs

import (
	"fmt"
	"regexp"
	"strings"
	// "github.com/fsouza/go-dockerclient"

	"github.com/pagodabox/nanobox-boxfile"
	"github.com/pagodabox/nanobox-golang-stylish"
	"github.com/pagodabox/nanobox-server/config"
	"github.com/pagodabox/nanobox-server/util"
)

//
type ServiceStart struct {
	deploy Deploy

	Boxfile boxfile.Boxfile
	EVars   map[string]string
	Success bool
	UID     string
}

//
func (j *ServiceStart) Process() {

	// var ci *docker.Container
	var err error

	j.Success = false

	util.LogInfo(stylish.Bullet(fmt.Sprintf("Starting %v...", j.UID)))

	createConfig := util.CreateConfig{Name: j.UID}

	image := regexp.MustCompile(`\d+`).ReplaceAllString(j.UID, "")
	if image == "web" || image == "worker" || image == "tcp" || image == "udp" {
		createConfig.Category = "code"
		image = "code"
	} else {
		createConfig.Category = "service"
	}

	fmt.Println(j.Boxfile)
	extra := strings.Trim(strings.Join([]string{j.Boxfile.VersionValue("version"), j.Boxfile.StringValue("stability")}, "-"), "-")
	if extra != "" {
		image = image + ":" + extra
	}

	createConfig.Image = "nanobox/" + image

	util.LogDebug(stylish.Bullet("image name: " + createConfig.Image))
	fmt.Println(createConfig)
	// start the container
	_, err = util.CreateContainer(createConfig)
	if err != nil {
		fmt.Println(err)
		util.HandleError(fmt.Sprintf("Failed to create %v", j.UID))
		util.UpdateStatus(&j.deploy, "errored")
		return
	}

	// payload
	payload := map[string]interface{}{
		"boxfile":     j.Boxfile.Parsed,
		"logtap_host": config.LogtapHost,
		"uid":         j.UID,

		// service hooks needed a reasonable default[:member][:schema][:meta][:ram]
		"member": map[string]interface{}{
			"schema": map[string]interface{}{
				"meta": map[string]interface{}{
					"ram": 128000000, // bytes
				},
			},
		},

		// service hooks need a reasonable default for [:ssh][:admin_key][:private_key]
		"ssh": map[string]interface{}{
			"admin_key": map[string]interface{}{
				"private_key": "notarealkey",
				"public_key":  "notarealkey",
			},
		},
	}

	// adds to the payload storage information if storage is required
	needsStorage := false
	storage := map[string]map[string]string{}
	for key, val := range j.EVars {
		matched, _ := regexp.MatchString(`NFS\d+_HOST`, key)
		if matched {
			needsStorage = true
			nfsUid := regexp.MustCompile(`_HOST`).ReplaceAllString(key, "")
			host := map[string]string{"host": val}
			storage[nfsUid] = host
		}
	}

	if needsStorage {
		payload["storage"] = storage
	}

	// run configure hook (blocking)
	if data, err := util.ExecHook("default-configure", j.UID, payload); err != nil {
		util.LogDebug("Failed Hook Output:\n%s\n", data)
		util.HandleError(fmt.Sprintf("ERROR configure %v\n", err))
		util.UpdateStatus(&j.deploy, "errored")
		return
	}

	// run start hook (blocking)
	if data, err := util.ExecHook("default-start", j.UID, payload); err != nil {
		util.LogDebug("Failed Hook Output:\n%s\n", data)
		util.HandleError(fmt.Sprintf("ERROR start %v\n", err))
		util.UpdateStatus(&j.deploy, "errored")
		return
	}

	// if we make it to the end it was a success!
	j.Success = true

	util.LogDebug("   [√] SUCCESS\n")
}
