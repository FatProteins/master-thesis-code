package util

import "sync"

type Pool[T any] struct {
	syncPool sync.Pool
}

func NewPool[T any]() *Pool[T] {
	return &Pool[T]{syncPool: sync.Pool{New: func() any { return new(T) }}}
}

func (pool *Pool[T]) Get() *T {
	return pool.syncPool.Get().(*T)
}

func (pool *Pool[T]) Put(value *T) {
	pool.syncPool.Put(value)
}
