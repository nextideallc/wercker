//   Copyright © 2016, 2018, Oracle and/or its affiliates.  All rights reserved.
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
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wercker/wercker/util"
	"golang.org/x/net/context"
)

type DockerSuite struct {
	*util.TestSuite
}

func TestDockerSuite(t *testing.T) {
	suiteTester := &DockerSuite{&util.TestSuite{}}
	suite.Run(t, suiteTester)
}

func (s *DockerSuite) TestPing() {
	ctx := context.Background()
	client := DockerOrSkip(ctx, s.T())
	_, err := client.Ping(ctx)
	s.Nil(err)
}

func (s *DockerSuite) TestGenerateDockerID() {
	id, err := GenerateDockerID()
	s.Require().NoError(err, "Unable to generate Docker ID")

	// The ID needs to be a valid hex value
	b, err := hex.DecodeString(id)
	s.Require().NoError(err, "Generated Docker ID was not a hex value")

	// The ID needs to be 256 bits
	s.Equal(256, len(b)*8)
}
