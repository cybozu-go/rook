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

	"github.com/rook/rook/pkg/util/sys"
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
	cmdArgs = utils.CommandArgs{Command: localPathPVForOSDCmd,
		CmdArgs: []string{installer.TestScratchDevice(), sys.DiskType, "create"}}
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

func (suite *OSDOnPVCSuite) createOSD(deviceType string) {
	cmdArgs := utils.CommandArgs{Command: localPathPVForOSDCmd,
		CmdArgs: []string{installer.TestScratchDevice(), deviceType, "create"}}
	cmdOut := utils.ExecuteCommand(cmdArgs)
	require.NoError(suite.T(), cmdOut.Err)

	_, err := suite.k8sh.Kubectl("-n", suite.namespace, "patch",
		"CephCluster", suite.op.installer.ClusterName,
		"--type=json", "-p", `[{"op": "replace", "path": "/spec/storage/storageClassDeviceSets/0/count", "value":1}]`)
	require.NoError(suite.T(), err)
	err = suite.k8sh.WaitForPodCount("app=rook-ceph-osd", suite.namespace, 1)
	require.Nil(suite.T(), err)
}

func (suite *OSDOnPVCSuite) deleteOSD(deviceType string) {
	_, err := suite.k8sh.Kubectl("-n", suite.namespace, "patch", "CephCluster", suite.op.installer.ClusterName,
		"--type=json", "-p", `[{"op": "replace", "path": "/spec/storage/storageClassDeviceSets/0/count", "value":0}]`)
	require.NoError(suite.T(), err)
	err = suite.k8sh.WaitForPodCount("app=rook-ceph-osd", suite.namespace, 0)
	require.Nil(suite.T(), err)
	_, err = suite.k8sh.Exec(suite.namespace, "rook-ceph-tools", "ceph", []string{"osd", "down", "osd.0"})
	require.Nil(suite.T(), err)
	_, err = suite.k8sh.Exec(suite.namespace, "rook-ceph-tools", "ceph", []string{"osd", "out", "osd.0"})
	require.Nil(suite.T(), err)
	_, err = suite.k8sh.Exec(suite.namespace, "rook-ceph-tools", "ceph", []string{"osd", "purge", "0", "--yes-i-really-mean-it"})
	require.Nil(suite.T(), err)
	_, err = suite.k8sh.Kubectl("-n", suite.namespace, "delete", "job", "-l", "app=rook-ceph-osd-prepare")
	require.Nil(suite.T(), err)
	_, err = suite.k8sh.Kubectl("-n", suite.namespace, "delete", "deployments", "rook-ceph-osd-0")
	require.Nil(suite.T(), err)
	_, err = suite.k8sh.Kubectl("-n", suite.namespace, "delete", "pvc", "-l", "ceph.rook.io/DeviceSet=set1")
	require.Nil(suite.T(), err)
	cmdArgs := utils.CommandArgs{Command: localPathPVForOSDCmd,
		CmdArgs: []string{installer.TestScratchDevice(), sys.DiskType, "delete"}}
	cmdOut := utils.ExecuteCommand(cmdArgs)
	require.NoError(suite.T(), cmdOut.Err)
}

// Test to make sure OSD on PVC works correctly
func (suite *OSDOnPVCSuite) TestCreatingOSDConfigurationOnPVC() {
	// We have an OSD on raw-disk-backed PVC here
	logger.Infof("Check if OSD on PVC backed by %q type device", sys.DiskType)
	checkIfRookClusterIsInstalled(suite.Suite, suite.k8sh, installer.SystemNamespace(suite.namespace), suite.namespace, 1)

	prevDeviceType := sys.DiskType
	for _, deviceType := range []string{sys.LVMType} {
		logger.Infof("Check if OSD on PVC backed by %q type device", deviceType)
		suite.deleteOSD(prevDeviceType)
		suite.createOSD(deviceType)
		prevDeviceType = deviceType
	}
}
