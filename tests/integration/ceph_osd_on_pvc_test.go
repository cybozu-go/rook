/*
Copyright 2020 The Rook Authors. All rights reserved.

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

package integration

import (
	"testing"

	"github.com/rook/rook/tests/framework/clients"
	"github.com/rook/rook/tests/framework/installer"
	"github.com/rook/rook/tests/framework/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	localPathPVForMonCmd = "tests/scripts/localPathPVForMon.sh"
	localPathPVForOSDCmd = "tests/scripts/localPathPVForOSD.sh"
)

// *************************************************************
// *** Major scenarios tested by the CephOSDOnPVCSuite ***
// Setup
// - via the CRD
// Monitors
// - Three mons
// OSDs
// - Bluestore running on a block device backed by a PVC
// *************************************************************
func TestCephOSDOnPVCSuite(t *testing.T) {
	if installer.SkipTestSuite(installer.CephTestSuite) {
		t.Skip()
	}
	v := version.MustParseSemantic(installer.KubeVersion())
	if !v.AtLeast(version.MustParseSemantic("v1.13.0")) {
		t.Skip()
	}
	s := new(OSDOnPVCSuite)
	defer func(s *OSDOnPVCSuite) {
		HandlePanics(recover(), s.op, s.T)
	}(s)
	suite.Run(t, s)
}

type OSDOnPVCSuite struct {
	suite.Suite
	helper    *clients.TestClient
	op        *TestCluster
	k8sh      *utils.K8sHelper
	namespace string
}

// createPVC creates a PVC for a OSD.
func (suite *OSDOnPVCSuite) createPVC() {
	cmdArgs := utils.CommandArgs{Command: localPathPVForMonCmd, CmdArgs: []string{}}
	cmdOut := utils.ExecuteCommand(cmdArgs)
	require.NoError(suite.T(), cmdOut.Err)
	cmdArgs = utils.CommandArgs{Command: localPathPVForOSDCmd, CmdArgs: []string{installer.TestScratchDevice()}}
	cmdOut = utils.ExecuteCommand(cmdArgs)
	require.NoError(suite.T(), cmdOut.Err)
}

// Deploy a Rook cluster
func (suite *OSDOnPVCSuite) SetupSuite() {
	suite.namespace = "osd-on-pvc-ns"
	osdOnPVCTestCluster := TestCluster{
		namespace:               suite.namespace,
		storeType:               "bluestore",
		storageClassName:        "manual",
		useHelm:                 false,
		usePVC:                  true,
		mons:                    3,
		rbdMirrorWorkers:        1,
		rookCephCleanup:         true,
		skipOSDCreation:         false,
		minimalMatrixK8sVersion: osdOnPVCSuiteMinimalTestVersion,
		rookVersion:             installer.VersionMaster,
		cephVersion:             installer.OctopusVersion,
	}
	suite.createPVC()
	suite.op, suite.k8sh = StartTestCluster(suite.T, &osdOnPVCTestCluster)
	suite.helper = clients.CreateTestClient(suite.k8sh, suite.op.installer.Manifests)
}

func (suite *OSDOnPVCSuite) AfterTest(suiteName, testName string) {
	suite.op.installer.CollectOperatorLog(suiteName, testName, installer.SystemNamespace(suite.namespace))
}

func (suite *OSDOnPVCSuite) TearDownSuite() {
	suite.op.Teardown()
}

// Test to make sure OSD on PVC works correctly
func (suite *OSDOnPVCSuite) TestCreatingOSDOnPVC() {
	// Check if a Rook cluster is deployed successfully
	checkIfRookClusterIsInstalled(suite.Suite, suite.k8sh, installer.SystemNamespace(suite.namespace), suite.namespace, 1)
}
