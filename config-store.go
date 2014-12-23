/*
Copyright 2014 Rohith All rights reserved.
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

package main

import (
	"flag"
	"syscall"
	"os/signal"
	"os"
	"time"

	"github.com/gambol99/config-store/store"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/golang/glog"
)

const (
	DEFAULT_MOUNT_POINT = "/data"
	DEFAULT_BACKEND     = "etcd://localhost:4001"
)

var (
	mount_point *string
)

func init() {
	mount_point = flag.String("mount", DEFAULT_MOUNT_POINT, "the mount of the fuse filesystem")
}

func main() {
	flag.Parse()
	/* step: lets check we have everything we* need */
	if *mount_point == "" {
		glog.Fatal("you have not specified a mount point to bind the filesystem to")
	}

	/* step: lets create the file system */
	filesystem, err := store.NewFuseKVFileSystem()
	if err != nil {
		glog.Errorf("Failed to create the K/V FileSystem, error: %s", err)
	} else {

	}
	nfs := pathfs.NewPathNodeFs( filesystem, nil)
	server, _, err := nodefs.MountRoot(
		*mount_point, nfs.Root(), &nodefs.Options{
			NegativeTimeout: 0,
			AttrTimeout:     time.Second,
			EntryTimeout:    time.Second,
			Owner:           &fuse.Owner{
				Uid: uint32(0),
				Gid: uint32(0)}})

	if err != nil {
		glog.Fatalf("Mount fail: %v\n", err)
	}
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-signalChannel
		glog.Infof("Recieved a kill signal, attempting to unmount and exit")
		server.Unmount()
		os.Exit(0)
	}()
	server.Serve()
}
