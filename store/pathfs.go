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

package store

import (
	"flag"
	"fmt"
	"strings"
	"time"
	"path/filepath"

	"github.com/gambol99/config-store/store/cache"
	"github.com/gambol99/config-store/store/config"
	"github.com/golang/glog"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

/*
Wasn't really sure how to implement this; etcd does provide a means to retrieve creation, modified
timestamps. Since i dont want to place it as a requirement of the k/v store, we could read the entire
configuration into a memory tree map and associate the metadata ... or we could use a map to track
changes - which is want i'm using at the moment
 */

type FuseKVFileSystem struct {
	/* the path file system interface we have to implement */
	pathfs.FileSystem
	/* the cache used by the fs */
	Cache cache.Cache
	/* the kv agent we are using */
	StoreKV config.KVStore
	/* the time we were created / initialized */
	BigBang time.Time
	/* a map of file name to last change event */
	NodeChanges map[string]time.Time
}

var backend_kv_url *string

const FUSE_VERBOSE_LEVEL = 7

func Verbose(message string, args ...interface{}) {
	glog.V(FUSE_VERBOSE_LEVEL).Infof(message, args)
}

func init() {
	backend_kv_url = flag.String( "kv", "etcd://127.0.0.1:4001", "the backend url for the key/value store" )
}

func (px *FuseKVFileSystem) NodeWatcher() error {
	updateChannel := make(chan config.NodeChange,0)
	stopChannel, err := px.StoreKV.Watch( "/", updateChannel )
	if err != nil {
		glog.Errorf("Unable to create a watch on root, error: %s", err )
		return err
	}
	go func() {
		for {
			/* step: we wait for an update */
			update := <- updateChannel
			Verbose("NodeWatcher() update: %s", update )
			switch update.Operation {
			case config.CHANGED:
				px.NodeChanges[update.Node.Path] = time.Now()
			case config.DELETED:
				if _, found := px.NodeChanges[update.Node.Path]; found {
					delete(px.NodeChanges,update.Node.Path)
				}
			}
			/* step: remove the node from the cache */
			px.CleanNode(update.Node.Path)
			px.CleanDir(update.Node.Path)
		}
		stopChannel <- true
	}()
	return nil
}

func (px *FuseKVFileSystem) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	/* step: delete the key pair */
	Verbose("Unlink() deleting the file: %s, context: %V", name, context)
	if err := px.StoreKV.Delete(name); err != nil {
		glog.Errorf("Failed to delete the key: %s, error: %s", name, err)
		return fuse.EPERM
	}
	return fuse.OK
}

func (px *FuseKVFileSystem) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	if name == "" {
		return &fuse.Attr{Mode: fuse.S_IFDIR | 0555}, fuse.OK
	}
	if node, err := px.CachedNode(name); err != nil {
		return nil, fuse.ENOENT
	} else {
		var attr fuse.Attr
		attr.Ctime = uint64(px.BigBang.Unix())
		attr.Mode = fuse.S_IFDIR|0665
		attr.Gid  = 0
		attr.Uid  = 0
		if _, found := px.NodeChanges[node.Path]; found {
			attr.Mtime = uint64(px.NodeChanges[node.Path].Unix())
		} else {
			attr.Mtime = uint64(px.BigBang.Unix())
		}
		if node.IsDir() {

		} else {
			attr.Mode = fuse.S_IFREG|0444
			attr.Size = uint64(len(node.Value))
		}
		return &attr, fuse.OK
	}
}

func (px *FuseKVFileSystem) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	Verbose("Rmdir() removing the directory: %s, context: %V", name, context)
	return fuse.EPERM
}

func (px *FuseKVFileSystem) Mkdir(name string, mode uint32, context *fuse.Context) fuse.Status {
	Verbose("Mkdir() path: %s, mode: %d, context: %V", name, mode, context)
	return fuse.EPERM
}

func (px *FuseKVFileSystem) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	Verbose("Open() name: %s, flags: %d, context: %V", name, flags, context)
	return NewKVFile(name, px.StoreKV), fuse.OK
}

func (px *FuseKVFileSystem) Create(name string, flags uint32, mode uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	Verbose("Open() name: %s, flags: %d, mode: %d, context: %V", name, flags, mode, context)
	return nil, fuse.EPERM
}

func (px *FuseKVFileSystem) OpenDir(name string, context *fuse.Context) (stream []fuse.DirEntry, status fuse.Status) {
	entries := []fuse.DirEntry{}
	/* step: get a list of the nodes under the path */
	Verbose("Opendir() key: %s", name )
	if nodes, err := px.CachedListing(name); err != nil {
		glog.Errorf("OpenDir() path: %s, context: %V, error: %s", name, context, err)
		return entries, fuse.EPERM
	} else {
		Verbose("OpenDir() nodes: %v", nodes)
		for _, node := range nodes {
			chunks := strings.Split(node.Path, "/")
			file := chunks[len(chunks) - 1]
			if node.IsDir() {
				entries = append(entries, fuse.DirEntry{Name: file, Mode: fuse.S_IFDIR })
			} else {
				entries = append(entries, fuse.DirEntry{Name: file, Mode: fuse.S_IFREG })
			}
		}
		return entries, fuse.OK
	}
}

func (px *FuseKVFileSystem) String() string {
	return fmt.Sprintf("FuseKVFileSystem(%v)", px.FileSystem)
}

const (
	SUFFIX_CACHE_NODE	 = "-node"
	SUFFIX_CACHE_LISTING = "-list"
)

func (px *FuseKVFileSystem) CleanNode(key string) {
	Verbose("CleanNode() removing the node: %s from cache", key )
	px.Cache.Delete( key + SUFFIX_CACHE_NODE )
}

func (px *FuseKVFileSystem) CleanDir(key string) {
	cacheKey := filepath.Dir(key) + SUFFIX_CACHE_LISTING
	Verbose("CleanDir() removing the parent key: %s", cacheKey )
	px.Cache.Delete(cacheKey)
}

func (px *FuseKVFileSystem) CachedNode(key string) (*config.Node,error) {
	var node *config.Node
	cacheKey := ""
	if key != ""  {
		cacheKey = "/"+key+SUFFIX_CACHE_NODE
		node, found := px.Cache.Get(cacheKey)
		if found {
			return node.(*config.Node), nil
		}
	}
	node, err := px.StoreKV.Get(key)
	if err != nil {
		glog.Errorf("GetAttr() failed get attribute, path: %s, error: %s", key, err)
		return nil, err
	}
	if key != "" {
		px.Cache.Set(cacheKey, node, 0)
	}
	return node, nil
}

func (px *FuseKVFileSystem) CachedListing(key string) ([]*config.Node,error) {
	cacheKey := "/"+key+SUFFIX_CACHE_LISTING
	if key != "" {
		nodes, _ := px.Cache.Get(cacheKey)
		if nodes != nil {
			return nodes.([]*config.Node), nil
		}
	}
	nodes, err := px.StoreKV.List(key)
	if err != nil || nodes == nil {
		return nil, err
	}
	if key != "" {
		px.Cache.Set(cacheKey, nodes, 0)
	}
	return nodes, nil
}

