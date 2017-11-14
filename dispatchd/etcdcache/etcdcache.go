package etcdcache

import (
	"context"
	"errors"
	"sync"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"
)

var etcdAPI = etcdclient.GetEtcdv3()

// EtcdCache stores etcd records in memory to do requests on
type EtcdCache struct {
	cache      map[string]*mvccpb.KeyValue
	cacheMutex sync.Mutex
}

// New gives a new empty EtcdCache
func New() *EtcdCache {
	return &EtcdCache{
		cache:      map[string]*mvccpb.KeyValue{},
		cacheMutex: sync.Mutex{},
	}
}

// NewForPrefix  returns a new EtcdCache prefilled with the keys of a specific prefix
func NewForPrefix(prefix string) (*EtcdCache, error) {
	cache := New()
	res, err := etcdAPI.Get(context.Background(), prefix, etcd.WithPrefix())
	if err != nil {
		return cache, err
	}
	for _, kv := range res.Kvs {
		cache.Put(kv)
	}

	return cache, nil
}

// Get gets a key out of the cache
func (e *EtcdCache) Get(key string) (*mvccpb.KeyValue, error) {
	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()
	if val, ok := e.cache[key]; ok {
		return val, nil
	}
	return nil, errors.New("Key not in cache")
}

// Put allows to add a new record
func (e *EtcdCache) Put(val *mvccpb.KeyValue) {
	e.cacheMutex.Lock()
	e.cache[string(val.Key)] = val
	e.cacheMutex.Unlock()
}

// GetAll returns all cached keysaa
func (e *EtcdCache) GetAll() []*mvccpb.KeyValue {
	out := []*mvccpb.KeyValue{}

	for _, val := range e.cache {
		out = append(out, val)
	}

	return out
}

// Invalidate removes a key from cache
func (e *EtcdCache) Invalidate(key string) {
	if _, exists := e.cache[key]; exists {
		delete(e.cache, key)
	}
}
