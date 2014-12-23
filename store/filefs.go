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
	"time"

	"github.com/gambol99/config-store/store/config"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/golang/glog"
)

type KVFile struct {
	/* The path / key of the file */
	Path string
	/* The value of the key */
	Value string
	/* The KVStore provider */
	StoreKV config.KVStore
}

func NewKVFile(path string, store config.KVStore) nodefs.File {
	Verbose("Creating K/V File, path: %s", path)
	file := new(KVFile)
	file.Path = path
	file.StoreKV = store
	return file
}

func (f *KVFile) String() string {
	return f.Path
}

func (f *KVFile) Read(buf []byte, off int64) (fuse.ReadResult, fuse.Status) {
	if node, err := f.StoreKV.Get(f.Path); err != nil {
		glog.Errorf("Read() file: %s failed to read, error: %s", f.Path, err)
		return nil, fuse.EIO
	} else {
		end := int(off) + int(len(buf))
		if end > len(node.Value) {
			end = len(node.Value)
		}
		data := []byte(node.Value)
		return fuse.ReadResultData(data[off:end]), fuse.OK
	}
}

/*
	We do not allow writing for the file. This should be handled at the K/V end and
	not by the client
*/
func (f *KVFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	Verbose("Write: file: %s, data: %V, off: %d", f.Path, data, off)
	return 0, fuse.EPERM
}

func (f *KVFile) Flush() fuse.Status {
	return fuse.OK
}

func (f *KVFile) Release() {

}

func (f *KVFile) GetAttr(attr *fuse.Attr) fuse.Status {
	if node, err := f.StoreKV.Get(f.Path); err != nil {
		glog.Errorf("GetAttr() Failed to get the key: %s, error: %s", f.Path, err)
		return fuse.EIO
	} else {
		attr.Mode = fuse.S_IFREG | 0444
		attr.Size = uint64(len(node.Value))
	}
	return fuse.OK
}

func (f *KVFile) Fsync(flags int) (code fuse.Status) {
	Verbose("Fsync() file: %s, flags: %d", f.Path, flags)
	return fuse.OK
}

func (f *KVFile) Utimens(atime *time.Time, mtime *time.Time) fuse.Status {
	Verbose("Utimens() file: %s, atime: %s, mtime: %s", f.Path, atime, mtime)
	return fuse.ENOSYS
}

func (f *KVFile) Truncate(size uint64) fuse.Status {
	Verbose("Truncate() file: %s, size: %d", f.Path, size)
	return fuse.OK
}

func (f *KVFile) Chown(uid uint32, gid uint32) fuse.Status {
	Verbose("Chown() uid: %s, gid: %d", uid, gid)
	return fuse.ENOSYS
}

func (f *KVFile) Chmod(perms uint32) fuse.Status {
	Verbose("Chmod() file: %s, perms:", f.Path, perms)
	return fuse.ENOSYS
}

func (f *KVFile) Allocate(off uint64, size uint64, mode uint32) (code fuse.Status) {
	Verbose("Chmod() file: %s, off: %d, sixe: %d, mode: %d", f.Path, off, size, mode)
	return fuse.OK
}

func (f *KVFile) SetInode(node *nodefs.Inode) {
	Verbose("SetInode() file: %s, node: %V", f.Path, node)
}

func (f *KVFile) InnerFile() nodefs.File {
	return nil
}
