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
package osd

import (
	"testing"

	rookv1 "github.com/rook/rook/pkg/apis/rook.io/v1"
	"github.com/rook/rook/pkg/clusterd"
	testexec "github.com/rook/rook/pkg/operator/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrepareDeviceSets(t *testing.T) {
	clientset := testexec.New(t, 1)
	context := &clusterd.Context{
		Clientset: clientset,
	}
	storageClass := "mysource"
	claim := v1.PersistentVolumeClaim{Spec: v1.PersistentVolumeClaimSpec{
		StorageClassName: &storageClass,
	}}
	deviceSet := rookv1.StorageClassDeviceSet{
		Name:                 "mydata",
		Count:                1,
		Portable:             true,
		VolumeClaimTemplates: []v1.PersistentVolumeClaim{claim},
		SchedulerName:        "custom-scheduler",
	}
	desired := rookv1.StorageScopeSpec{StorageClassDeviceSets: []rookv1.StorageClassDeviceSet{deviceSet}}
	cluster := &Cluster{
		context:        context,
		DesiredStorage: desired,
		Namespace:      "testns",
	}

	config := &provisionConfig{}
	volumeSources := cluster.prepareStorageClassDeviceSets(config)
	assert.Equal(t, 1, len(volumeSources))
	assert.Equal(t, 0, len(config.errorMessages))
	assert.Equal(t, "mydata", volumeSources[0].Name)
	assert.Equal(t, "data", volumeSources[0].Type)
	assert.True(t, volumeSources[0].Portable)
	assert.Equal(t, "custom-scheduler", volumeSources[0].SchedulerName)

	// Verify that the PVC has the expected generated name with the default of "data" in the name
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(cluster.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pvcs.Items))
	assert.Equal(t, "mydata-data-0-", pvcs.Items[0].GenerateName)
	assert.Equal(t, cluster.Namespace, pvcs.Items[0].Namespace)
}
