package cache

import "k8s.io/client-go/tools/cache"

func NewIndexer(keyFunc cache.KeyFunc, indexers cache.Indexers) cache.Indexer {
	return &dummyIndexer{}
}

type dummyIndexer struct{}

func (d dummyIndexer) Add(obj interface{}) error {
	return nil
}

func (d dummyIndexer) Update(obj interface{}) error {
	return nil
}

func (d dummyIndexer) Delete(obj interface{}) error {
	return nil
}

func (d dummyIndexer) List() []interface{} {
	return nil
}

func (d dummyIndexer) ListKeys() []string {
	return nil
}

func (d dummyIndexer) Get(_ interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (d dummyIndexer) GetByKey(_ string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (d dummyIndexer) Replace(_ []interface{}, _ string) error {
	return nil
}

func (d dummyIndexer) Resync() error {
	return nil
}

func (d dummyIndexer) Index(_ string, _ interface{}) ([]interface{}, error) {
	return nil, nil
}

func (d dummyIndexer) IndexKeys(_, _ string) ([]string, error) {
	return nil, nil
}

func (d dummyIndexer) ListIndexFuncValues(_ string) []string {
	return nil
}

func (d dummyIndexer) ByIndex(_, _ string) ([]interface{}, error) {
	return nil, nil
}

func (d dummyIndexer) GetIndexers() cache.Indexers {
	return nil
}

func (d dummyIndexer) AddIndexers(_ cache.Indexers) error {
	return nil
}
