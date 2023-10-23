package consensus

import "context"

type ConsensusClient interface {
	SubscribeToChanges(ctx context.Context) (<-chan interface{}, error)
	GetKV(addKV func(string, string)) error
	StoreKV(key, value string) error
	DeleteKey(key string) error
}
