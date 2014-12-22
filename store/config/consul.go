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

package config

import (
	"flag"
	"net/url"
	"time"

	consulapi "github.com/armon/consul-api"
	"github.com/golang/glog"
)

var consul_datacenter, consul_token *string

func init() {
	consul_datacenter = flag.String("-consul-dc", "DC1", "the consul datacenter parameter")
	consul_token = flag.String("-consul-token", "", "the securirty token used for write operations")
}

type ConsulClient struct {
	/* the consul client */
	Client *consulapi.Client
	/* the write options for client */
	WriteOptions *consulapi.WriteOptions
}

func NewConsulStoreClient(uri *url.URL) (KVStore, error) {
	glog.Infof("Creating a new Consul K/V Client, url: %s", uri)
	config := consulapi.DefaultConfig()
	config.Address = uri.Host
	client, err := consulapi.NewClient(config)
	if err != nil {
		glog.Errorf("Failed to create the Consul Clinet, error: %s", err)
		return nil, err
	}
	kv := new(ConsulClient)
	kv.Client = client
	kv.WriteOptions = &consulapi.WriteOptions{
		Datacenter: *consul_datacenter,
		Token:      *consul_token}
	return kv, nil
}

func (r *ConsulClient) Get(key string) (*Node,error) {
	if response, _, err := r.Client.KV().Get(key, &consulapi.QueryOptions{}); err != nil {
		glog.Errorf("Get() failed to get key: %s, error: %s", key, err)
		return nil, err
	} else {
		return &Node{
			Path: key,
			Value: string(response.Value[:]),
			Directory: true}, nil
	}
}

func (r *ConsulClient) Set(key string, value string) error {
	Verbose("Set() key: %s, value: %s", key, value)
	_, err := r.Client.KV().Put(&consulapi.KVPair{Key: key, Value: []byte(value)}, nil)
	if err != nil {
		glog.Errorf("Set() failed to set key: %s, error: %s", key, err)
		return err
	}
	return nil
}

func (r *ConsulClient) Delete(key string) error {
	Verbose("Delete() deleting the key: %s", key)
	_, err := r.Client.KV().Delete(key, r.WriteOptions)
	if err != nil {
		glog.Errorf("Delete() failed to delete key: %s, error: %s", key, err)
		return err
	}
	return nil
}

func (r *ConsulClient) RemovePath(path string) error {
	Verbose("RemovePath() deleting the path: %s", path)
	if _, err := r.Client.KV().DeleteTree(path, r.WriteOptions); err != nil {
		glog.Errorf("RemovePath() failed to remove path: %s, error: %s", path, err)
		return err
	}
	return nil
}

func (r *ConsulClient) Mkdir(path string) error {
	Verbose("Mkdir() path: %s", path)
	return nil
}

func (r *ConsulClient) List(path string) ([]*Node, error) {
	Verbose("List() path: %s", path)
	if response, _, err := r.Client.KV().List(path, &consulapi.QueryOptions{}); err != nil {
		glog.Errorf("List() failed to list path: %s, error: %s", path, err)
		return nil, err
	} else {
		list := make([]*Node, 0)
		for _, pair := range response {
			node := &Node{
				Path: pair.Key,
				Directory: true,
				Value: string(pair.Value[:])}
			list = append(list, node)
		}
		return list, nil
	}
}

func (r *ConsulClient) Watch(key string, updateChannel chan NodeChange) (chan bool,error) {
	Verbose("Watch() key: %s, channel: %V", key, updateChannel)
	stopChannel := make(chan bool,0)
	killOffWatch := false
	go func() {
		/* step: wait for the shutdown signal */
		<-stopChannel
		glog.V(3).Infof("Watch() killing off the watch on key: %s", key)
		killOffWatch = true
	}()
	waitIndex := uint64(0)
	go func() {
		for {
			if killOffWatch {
				glog.V(3).Infof("Watch() exitting the watch on key: %s", key)
				break
			}
			response, meta, err := r.Client.KV().Get(key, &consulapi.QueryOptions{WaitIndex: waitIndex})
			if err != nil {
				glog.Errorf("Watch() error attempting to watch the key: %s, error: %s", key, err)
				time.Sleep(3 * time.Second)
				continue
			}
			if killOffWatch {
				continue
			}
			if waitIndex == meta.LastIndex {
				Verbose("Watch() key: %s, skipping the change, indexes are the same", key)
				continue
			}
			/* step: pass the change upstream */
			Verbose("Watch() sending the change for key: %s upstream", key)
			updateChannel <- r.GetNodeEvent(response)
		}
	}()
	return stopChannel, nil
}

func (r *ConsulClient) GetNodeEvent(response *consulapi.KVPair) (event NodeChange) {
	event.Node.Path = response.Key
	event.Node.Value = string(response.Value[:])
	event.Operation = CHANGED
	return
}
