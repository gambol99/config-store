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
	"errors"
	"net/url"

	"github.com/gambol99/config-store/cache"
	"github.com/gambol99/config-store/store/config"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/golang/glog"
)

func NewFuseKVFileSystem() (pathfs.FileSystem, error) {
	glog.Infof("Creating a new K/V FileSystem, backend: %s", *backend_kv_url)
	/* step: parse the url and make sure it's valid */
	uri, err := url.Parse(*backend_kv_url)
	if err != nil {
		glog.Errorf("Failed to parse the url: %s, probably invalid, error: %s", *backend_kv_url, err)
		return nil, err
	}
	/* step: create a backend K/V client */
	var kv_agent config.KVStore
	switch uri.Scheme {
	case "etcd":
		kv_agent, err = config.NewEtcdStoreClient(uri)
	case "consul":
		kv_agent, err = config.NewConsulStoreClient(uri)
	default:
		glog.Errorf("Invalid backend url: %s, unsupported provider, please check usage", *backend_kv_url)
		return nil, errors.New("Unsupported backend k/v provider: " + *backend_kv_url)
	}
	/* step: validate the error */
	if err != nil {
		glog.Errorf("Failed to create the K/V agent for filesystem, error: %s", err)
		return nil, err
	}
	fs := &FuseKVFileSystem{pathfs.NewDefaultFileSystem(),cache.NewCacheStore(),kv_agent}
	return fs, nil
}
