// Copyright (c) 2014 Pagoda Box Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public License,
// v. 2.0. If a copy of the MPL was not distributed with this file, You can
// obtain one at http://mozilla.org/MPL/2.0/.

package util

//
import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"io"

	"github.com/fsouza/go-dockerclient"
	"github.com/pagodabox/nanobox-server/config"
	docksig "github.com/docker/docker/pkg/signal"
)

func CreateEnterContainer(name string, cmd []string) (*docker.Container, error) {
	cConfig := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			OpenStdin:       true,
			Env:             []string{`PATH=/data/sbin:/data/bin:/opt/gonano/sbin:/opt/gonano/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin`},
			Tty:             true,
			Labels:          map[string]string{"enter": "true", "uid": name},
			NetworkDisabled: false,
			WorkingDir:      "/code",
			Image:           "nanobox/build",
			Cmd:             cmd,
		},
		HostConfig: &docker.HostConfig{
			Binds: []string{
				"/mnt/sda/var/nanobox/deploy/:/data/",
				"/vagrant/code/" + config.App + "/:/code/",
			},
			Privileged: true,
		},
	}

	return createContainer(cConfig)
}

// CreateBuildContainer
func CreateBuildContainer(name string) (*docker.Container, error) {
	cConfig := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Tty:             true,
			Labels:          map[string]string{"build": "true", "uid": name},
			NetworkDisabled: false,
			Image:           "nanobox/build",
			Cmd:             []string{"/bin/sleep", "365d"},
		},
		HostConfig: &docker.HostConfig{
			Binds: []string{
				"/mnt/sda/var/nanobox/cache/:/mnt/cache/",
				"/mnt/sda/var/nanobox/deploy/:/mnt/deploy/",
				"/mnt/sda/var/nanobox/build/:/mnt/build/",

				"/vagrant/code/" + config.App + "/:/share/code/:ro",
				"/vagrant/engines/:/share/engines/:ro",
			},
			Privileged: true,
		},
	}

	return createContainer(cConfig)
}

// CreateCodeContainer
func CreateCodeContainer(name string) (*docker.Container, error) {
	cConfig := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Tty:             true,
			Labels:          map[string]string{"code": "true", "uid": name},
			NetworkDisabled: false,
			Image:           "nanobox/code",
		},
		HostConfig: &docker.HostConfig{
			Binds: []string{
				"/mnt/sda/var/nanobox/deploy/:/data/",
				"/mnt/sda/var/nanobox/build/:/code/:ro",
			},
			Privileged: true,
		},
	}

	return createContainer(cConfig)
}

// CreateServiceContainer
func CreateServiceContainer(name, image string) (*docker.Container, error) {
	cConfig := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Tty:             true,
			Labels:          map[string]string{"service": "true", "uid": name},
			NetworkDisabled: false,
			Image:           image,
		},
		HostConfig: &docker.HostConfig{
			Binds:      []string{},
			Privileged: true,
		},
	}

	return createContainer(cConfig)
}

// createContainer
func createContainer(cConfig docker.CreateContainerOptions) (*docker.Container, error) {

	// LogInfo("CREATE CONTAINER! %#v", cConfig)

	//
	if !ImageExists(cConfig.Config.Image) {
		if err := dockerClient().PullImage(docker.PullImageOptions{Repository: cConfig.Config.Image}, docker.AuthConfiguration{}); err != nil {
			return nil, err
		}
	}

	// create container
	container, err := dockerClient().CreateContainer(cConfig)
	if err != nil {
		LogError("Unable to create Container %v", err)
		return nil, err
	}

	if err := StartContainer(container.ID); err != nil {
		return nil, err
	}

	return container, nil
}

// Start
func StartContainer(id string) error {
	return dockerClient().StartContainer(id, &docker.HostConfig{
		Privileged: true,
	})
}

func AttachToContainer(id string, in io.Reader, out io.Writer, err io.Writer) error {
	attachConfig := docker.AttachToContainerOptions{
	    Container: id,
	    InputStream: in,
	    OutputStream: out,
	    ErrorStream: err,
	    Stream: true,
	    Stdin: true,
	    Stdout: true,
	    Stderr: true,
	    RawTerminal: true,
	}
	return dockerClient().AttachToContainer(attachConfig)
}

func KillContainer(id, sig string) error {
	return dockerClient().KillContainer(docker.KillContainerOptions{ID: id, Signal: docker.Signal(docksig.SignalMap[sig])})
}

func ResizeContainerTTY(id string, height, width int) error {
	return dockerClient().ResizeContainerTTY(id, height, width)
}

func WaitContainer(id string) (int, error) {
	return dockerClient().WaitContainer(id)
}

// RemoveContainer
func RemoveContainer(id string) error {
	// if _, err := dockerClient().InspectContainer(id); err != nil {
	// 	return err
	// }

	if err := dockerClient().StopContainer(id, 0); err != nil {
		// return err
	}

	return dockerClient().RemoveContainer(docker.RemoveContainerOptions{ID: id, RemoveVolumes: false, Force: true})
}

// InspectContainer
func InspectContainer(id string) (*docker.Container, error) {
	return dockerClient().InspectContainer(id)
}

// GetContainer
func GetContainer(name string) (*docker.Container, error) {
	containers, err := ListContainers()
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.Name == name || container.Name == ("/"+name) || container.ID == name {
			return InspectContainer(container.ID)
		}
	}
	return nil, errors.New("not found")
}

// ListContainers
func ListContainers(labels ...string) ([]*docker.Container, error) {
	rtn := []*docker.Container{}

	apiContainers, err := dockerClient().ListContainers(docker.ListContainersOptions{All: true, Size: false})
	if len(labels) == 0 || err != nil {
		for _, apiContainer := range apiContainers {
			container, _ := InspectContainer(apiContainer.ID)
			if container != nil {
				rtn = append(rtn, container)
			}
		}
		return rtn, err
	}

	for _, apiContainer := range apiContainers {
		container, _ := InspectContainer(apiContainer.ID)
		if container != nil {
			for _, label := range labels {
				if container.Config.Labels[label] == "true" {
					rtn = append(rtn, container)
				}
			}
		}
	}

	return rtn, nil
}

// Exec
func ExecInContainer(container string, args ...string) ([]byte, error) {
	opts := docker.CreateExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          args,
		Container:    container,
	}
	exec, err := dockerClient().CreateExec(opts)

	if err != nil {
		return []byte{}, err
	}
	var b bytes.Buffer
	err = dockerClient().StartExec(exec.ID, docker.StartExecOptions{OutputStream: &b, ErrorStream: &b})
	// LogDebug("execincontainer: %s\n", b.Bytes())
	results, err := dockerClient().InspectExec(exec.ID)
	// LogDebug("execincontainer results: %+v\n", results)
	if err != nil {
		return b.Bytes(), err
	}
	if results.ExitCode != 0 {
		return b.Bytes(), errors.New(fmt.Sprintf("Bad Exit Code (%d)", results.ExitCode))
	}
	return b.Bytes(), err
}

// Run
func RunInContainer(container, img string, args ...string) ([]byte, error) {

	// build the initial command, and then iterate over any additional arguments
	// that are passed in as commands adding them the the final command
	cmd := []string{"run", "--rm", container, img}
	for _, a := range args {
		cmd = append(cmd, a)
	}

	return exec.Command("docker", cmd...).Output()
}

// ImageExists
func ImageExists(name string) bool {
	images, err := dockerClient().ListImages(docker.ListImagesOptions{All: true})
	if err != nil {
		return false
	}
	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == name+":latest" {
				return true
			}
		}
	}

	return false
}

func InstallImage(image string) error {
	if err := dockerClient().PullImage(docker.PullImageOptions{Repository: image}, docker.AuthConfiguration{}); err != nil {
		return err
	}

	return nil
}

// PullImage
func UpdateAllImages() error {
	images, err := dockerClient().ListImages(docker.ListImagesOptions{All: true})
	if err != nil {
		return err
	}
	for _, image := range images {
		for _, tag := range image.RepoTags {
			err := UpdateImage(tag)
			if err != nil {
				return err
			}
		}
	}
	config.Mist.Publish([]string{"update"}, `{"model":"Update", "action":"update", "document":"{\"id\":\"1\", \"status\":\"complete\"}"}`)
	return nil
}

func UpdateImage(image string) error {
	config.Mist.Publish([]string{"update"}, fmt.Sprintf(`{"model":"Update", "action":"update", "document":"{\"id\":\"1\", \"status\":\"pulling image for %s\"}"}`, image))
	if err := dockerClient().PullImage(docker.PullImageOptions{Repository: image}, docker.AuthConfiguration{}); err != nil {
		return err
	}

	return nil
}

// dockerClient
func dockerClient() *docker.Client {
	d, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		config.Log.Error(err.Error())
	}
	return d
}
