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

package cache

import (
	"sync"
	"time"

	"github.com/golang/glog"
)

type CacheStore interface {
	/* retrieve an entry from the cache */
	Get(string) interface{}
	/* set a entry in the cache */
	Set(string, interface{},time.Duration)
	/* check if a key exist in the cache */
	Exists(string) bool
	/* get the size of the cache */
	Size() int
	/* flush the cache all entries */
	Flush() error
}

type CachedItem struct {
	/* the unix timestamp the item is expiring in */
	Expiring    int64
	/* the data we are holding */
	Data 		interface {}
}

type Cache struct {
	/* locking required for the cache */
	sync.RWMutex
	/* the cache map */
	Items map[string]*CachedItem
}

func NewCacheStore() CacheStore {
	glog.Infof("Creating a new cache store")
	return &Cache{Items: make(map[string]*CachedItem)}
}

func (c *Cache) Get(key string) interface{} {
	c.RLock()
	defer c.RUnlock()
	glog.V(9).Infof("Get() key: %s", key)
	if item, found := c.Items[key]; found {
		return item.Data
	}
	return nil
}

func (c *Cache) Set(key string, item interface{}, ttl time.Duration) {
	c.Lock()
	defer c.Unlock()
	expiration_time := int64(0)
	if ttl != 0 {
		expiration_time = (time.Now().Unix()) + int64(ttl.Seconds())
	}
	glog.V(9).Infof("Set() key: %s, value: %V, expiration: %d", key, item, expiration_time )
	c.Items[key] = &CachedItem{expiration_time,item}
}

func (c *Cache) Exists(key string) (found bool) {
	c.RLock()
	defer c.RUnlock()
	_, found = c.Items[key]
	return
}

func (c *Cache) Flush() error {
	c.Lock()
	defer c.Unlock()
	c.Items = make(map[string]*CachedItem)
	return nil
}

func (c *Cache) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.Items)
}
