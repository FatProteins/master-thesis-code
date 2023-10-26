package consensus

import "context"

type ConsensusClient interface {
	GetStatus() (NodeStatus, error)
	SubscribeToChanges(ctx context.Context) (<-chan []string, error)
	GetKV(addKV func(string, string)) error
	StoreKV(key, value string) error
	DeleteKey(key string) error
}

type NodeStatus struct {
	NodeId       string
	Leader       string
	Term         uint64
	Index        uint64
	AppliedIndex uint64
}
