#!/bin/bash

lsblk

sudo rm -rf /var/lib/rook/rook-integration-test
sudo mkdir -p /var/lib/rook/rook-integration-test/mon1 /var/lib/rook/rook-integration-test/mon2 /var/lib/rook/rook-integration-test/mon3

node_name=$(kubectl get nodes -o jsonpath={.items[*].metadata.name})

kubectl label nodes ${node_name} rook.io/has-disk=true

kubectl delete pv -l type=local-mon

cat <<eof | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: local-mon1
  labels:
    type: local-mon
spec:
  storageClassName: manual 
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce 
  persistentVolumeReclaimPolicy: Retain
  volumeMode: Filesystem
  local:
    path: "/var/lib/rook/rook-integration-test/mon1"
  nodeAffinity:
      required:
        nodeSelectorTerms:
          - matchExpressions:
              - key: rook.io/has-disk
                operator: In
                values:
                - "true"
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: local-mon2
  labels:
    type: local-mon
spec:
  storageClassName: manual 
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce 
  persistentVolumeReclaimPolicy: Retain
  volumeMode: Filesystem
  local:
    path: "/var/lib/rook/rook-integration-test/mon2"
  nodeAffinity:
      required:
        nodeSelectorTerms:
          - matchExpressions:
              - key: rook.io/has-disk
                operator: In
                values:
                - "true"
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: local-mon3
  labels:
    type: local-mon
spec:
  storageClassName: manual 
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce 
  persistentVolumeReclaimPolicy: Retain
  volumeMode: Filesystem
  local:
    path: "/var/lib/rook/rook-integration-test/mon3"
  nodeAffinity:
      required:
        nodeSelectorTerms:
          - matchExpressions:
              - key: rook.io/has-disk
                operator: In
                values:
                - "true"
---
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: manual
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
eof
