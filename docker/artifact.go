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
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/wercker/wercker/core"
	"github.com/wercker/wercker/util"
)

// Set upper limit that we can store
const maxArtifactSize = 5000 * 1024 * 1024 // in bytes

// Artificer collects artifacts from containers and uploads them.
type Artificer struct {
	options       *core.PipelineOptions
	dockerOptions *Options
	logger        *util.LogEntry
	store         core.Store
}

// NewArtificer returns an Artificer
func NewArtificer(options *core.PipelineOptions, dockerOptions *Options) *Artificer {
	logger := util.RootLogger().WithField("Logger", "Artificer")

	var store core.Store
	if options.Store == "s3" {
		store = core.NewS3Store(options.AWSOptions)
	} else if options.Store == "oci" {
		store = core.NewOciStore(options.OciOptions)
	}

	return &Artificer{
		options:       options,
		dockerOptions: dockerOptions,
		logger:        logger,
		store:         store,
	}
}

// Collect an artifact from the container, if it doesn't have any files in
// the tarball return util.ErrEmptyTarball
func (a *Artificer) Collect(artifact *core.Artifact) (*core.Artifact, error) {
	client, _ := NewDockerClient(a.dockerOptions)

	if err := os.MkdirAll(filepath.Dir(artifact.HostPath), 0755); err != nil {
		return nil, err
	}

	outputFile, err := os.Create(artifact.HostTarPath)
	defer outputFile.Close()
	if err != nil {
		return nil, err
	}

	dfc := NewDockerFileCollector(client, artifact.ContainerID)
	archive, errs := dfc.Collect(artifact.GuestPath)
	archive.Tee(outputFile)

	select {
	case err = <-errs:
	// TODO(termie): I hate this, but docker command either fails right away
	//               or we don't care about it, needs to be replaced by some
	//               sort of cancellable context
	case <-time.After(1 * time.Second):
		err = <-archive.Multi(filepath.Base(artifact.GuestPath), artifact.HostPath, maxArtifactSize)
	}

	if err != nil {
		if err == util.ErrEmptyTarball {
			return nil, err
		}
		return nil, err
	}
	return artifact, nil
}

// Upload an artifact to S3
func (a *Artificer) Upload(artifact *core.Artifact) error {
	return a.store.StoreFromFile(&core.StoreFromFileArgs{
		Path:        artifact.HostTarPath,
		Key:         artifact.RemotePath(),
		ContentType: artifact.ContentType,
		MaxTries:    3,
		Meta:        artifact.Meta,
	})
}

// DockerFileCollector impl of FileCollector
type DockerFileCollector struct {
	client      *DockerClient
	containerID string
	logger      *util.LogEntry
}

// NewDockerFileCollector constructor
func NewDockerFileCollector(client *DockerClient, containerID string) *DockerFileCollector {
	return &DockerFileCollector{
		client:      client,
		containerID: containerID,
		logger:      util.RootLogger().WithField("Logger", "DockerFileCollector"),
	}
}

// Collect grabs a path and returns an archive containing stream along with
// an error channel to select on
func (fc *DockerFileCollector) Collect(path string) (*util.Archive, chan error) {
	pipeReader, pipeWriter := io.Pipe()

	opts := docker.DownloadFromContainerOptions{
		OutputStream: pipeWriter,
		Path:         path,
	}

	errs := make(chan error)

	go func() {
		defer close(errs)
		if err := fc.client.DownloadFromContainer(fc.containerID, opts); err != nil {
			switch err.(type) {
			case *docker.Error:
				derr := err.(*docker.Error)
				if derr.Status == 500 && strings.HasPrefix(derr.Message, "Could not find the file") {
					errs <- util.ErrEmptyTarball
				}
			default:
				errs <- err
			}
		}
		pipeWriter.Close()
	}()

	return util.NewArchive(pipeReader), errs
}
