#!/bin/bash

function create_osd() {
  lsblk
  kubectl delete pv -l type=local-osd

  if [ $device_type = lvm ] ; then
    test_vg=rook-integration-test-vg
    test_lv=rook-integration-test-lv
    sudo pvcreate "${test_scratch_device}"
    sudo vgcreate "${test_vg}" "${test_scratch_device}"
    sudo lvcreate -l 100%FREE -n "${test_lv}" "${test_vg}"
    test_device=/dev/"${test_vg}"/"${test_lv}"
  fi

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
    storage: 9Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  volumeMode: Block
  local:
    path: "${test_device}"
  nodeAffinity:
      required:
        nodeSelectorTerms:
          - matchExpressions:
              - key: rook.io/has-disk
                operator: In
                values:
                - "true"
eof
}

function delete_osd() {
  lsblk
  kubectl delete pv -l type=local-osd

  if [ $device_type = lvm ] ; then
    sudo vgremove --yes --force rook-integration-test-vg
    sudo pvremove --yes --force ${test_scratch_device}
  fi

  sudo dd if=/dev/zero of=${test_scratch_device} bs=1M count=100 oflag=dsync,direct
}

test_scratch_device=/dev/xvdc
if [ $# -lt 3 ] ; then
  echo "usage: $0 <scratch device> <device type> <create|delete>" >&2
  exit 1
fi

test_scratch_device=$1
if [ ! -b "${test_scratch_device}" ] ; then
  echo "scratch device should be block device: ${test_scratch_device}" >&2
  exit 2
fi

device_type=$2
test_device=${test_scratch_device}
case "${device_type}" in
  disk|lvm)
    ;;
  *)
    echo "invalid device type: '$2'" >&2
    exit 3
    ;;
esac

command=$3
case "${command}" in
  create)
    create_osd
    ;;
  delete)
    delete_osd
    ;;
  *)
    echo "invalid command type: '$3'" >$2
    exit 4
    ;;
esac