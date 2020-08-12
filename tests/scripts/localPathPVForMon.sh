#!/bin/bash

lsblk

sudo rm -rf /var/lib/rook/rook-integration-test
sudo mkdir -p /var/lib/rook/rook-integration-test/mon

node_name=$(kubectl get nodes -o jsonpath={.items[*].metadata.name})

kubectl label nodes ${node_name} rook.io/has-disk=true

kubectl delete pv -l type=local-mon

cat <<eof | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: local-mon
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
    path: "/var/lib/rook/rook-integration-test/mon"
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
