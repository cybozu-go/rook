#!/bin/bash

test_scratch_device=/dev/xvdc
if [ $# -ge 1 ] ; then
  test_scratch_device=$1
fi

if [ ! -b "${test_scratch_device}" ] ; then
  echo "invalid scratch device name: ${test_scratch_device}" >&2
  exit 1
fi

lsblk

kubectl delete pv -l type=local-osd

cat <<eof | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: local-osd
  labels:
    type: local-osd
spec:
  storageClassName: manual
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  volumeMode: Block
  local:
    path: "${test_scratch_device}"
  nodeAffinity:
      required:
        nodeSelectorTerms:
          - matchExpressions:
              - key: rook.io/has-disk
                operator: In
                values:
                - "true"
eof
