#!/bin/bash
#
#   Author: Rohith (gambol99@gmail.com)
#   Date: 2014-12-23 12:30:27 +0000 (Tue, 23 Dec 2014)
#
#  vim:ts=2:sw=2:et
#

VERBOSE=${VERBOSE:-1}
STORE=${STORE:-'etcd://localhost:4001'}
mkdir -p /store
echo "Starting the Configuration Store Filesystem: kv: ${STORE}, mount: ${MOUNTPOINT}"
/bin/config-store -kv=${STORE} -mount=/store -logtostderr=true -v=${VERBOSE}
