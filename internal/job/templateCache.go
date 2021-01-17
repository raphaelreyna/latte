package job

import (
	"sync"
	lru "github.com/hashicorp/golang-lru"
)

type TemplateCache struct {
	sync.Mutex
	cache *lru.Cache
}

func NewTemplateCache(n int) (*TemplateCache, error) {
	var err error
	tc := &TemplateCache{}
	if tc.cache, err = lru.New(n); err != nil {
		return nil, err
	}

	return tc, nil
}

func (tc *TemplateCache) Get(key string) (interface{}, bool) {
	return tc.cache.Get(key)
}

func (tc *TemplateCache) Add(key string, val interface{}) bool {
	return tc.cache.Add(key, val)
}
