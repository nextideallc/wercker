//   Copyright 2016 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package dockerlocal

import (
	"os"
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/wercker/wercker/util"
	"golang.org/x/net/context"
)

// DockerOrSkip checks for a docker container and skips the test
// if one is not available
func DockerOrSkip(t *testing.T) *DockerClient {
	if os.Getenv("SKIP_DOCKER_TEST") == "true" {
		t.Skip("$SKIP_DOCKER_TEST=true, skipping test")
		return nil
	}

	client, err := NewDockerClient(MinimalDockerOptions())
	err = client.Ping()
	if err != nil {
		t.Skip("Docker not available, skipping test")
		return nil
	}
	return client
}

func MinimalDockerOptions() *Options {
	ctx := context.Background()
	opts := &Options{}
	guessAndUpdateDockerOptions(ctx, opts, util.NewEnvironment(os.Environ()...))
	return opts
}

type ContainerRemover struct {
	*docker.Container
	client *DockerClient
}

func TempBusybox(client *DockerClient) (*ContainerRemover, error) {
	_, err := client.InspectImage("alpine")
	if err != nil {
		options := docker.PullImageOptions{
			Repository: "alpine",
			Tag:        "3.1",
		}

		err = client.PullImage(options, docker.AuthConfiguration{})
		if err != nil {
			return nil, err
		}
	}

	container, err := client.CreateContainer(
		docker.CreateContainerOptions{
			Name: "temp-busybox",
			Config: &docker.Config{
				Image:           "alpine:3.1",
				Tty:             false,
				OpenStdin:       true,
				Cmd:             []string{"/bin/sh"},
				AttachStdin:     true,
				AttachStdout:    true,
				AttachStderr:    true,
				NetworkDisabled: true,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &ContainerRemover{Container: container, client: client}, nil
}

func (c *ContainerRemover) Remove() {
	if c == nil {
		return
	}
	c.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            c.Container.ID,
		RemoveVolumes: true,
	})
}
