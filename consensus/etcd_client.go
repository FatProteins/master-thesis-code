package consensus

import (
	"context"
	"fmt"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

var etcdClientLogger = daLogger.NewLogger("etcdclient")

type EtcdClient struct {
	internalEndpoint string
	internalClient   *clientv3.Client
}

func NewEtcdClient(ctx context.Context, mainEndpoint string) *EtcdClient {
	mainClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{mainEndpoint},
		DialTimeout: 10 * time.Second,
		Context:     ctx,
	})
	if err != nil {
		panic(err)
	}

	return &EtcdClient{internalEndpoint: mainEndpoint, internalClient: mainClient}
}

func (c *EtcdClient) GetStatus() (NodeStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	statusResp, err := c.internalClient.Status(ctx, c.internalEndpoint)
	if err != nil {
		return NodeStatus{}, err
	}
	return NodeStatus{
		NodeId:       fmt.Sprintf("%x", statusResp.Header.MemberId),
		Leader:       fmt.Sprintf("%x", statusResp.Leader),
		Term:         statusResp.RaftTerm,
		Index:        statusResp.RaftIndex,
		AppliedIndex: statusResp.RaftAppliedIndex,
	}, nil
}

func (c *EtcdClient) SubscribeToChanges(ctx context.Context) (<-chan []string, error) {
	ch := make(chan []string)
	watchChan := c.internalClient.Watch(ctx, "", clientv3.WithPrefix())

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case resp, ok := <-watchChan:
				if !ok {
					close(ch)
					return
				}

				var changes []string
				for _, event := range resp.Events {
					var description string
					if event.Type == mvccpb.PUT {
						description = fmt.Sprintf("%s on Key '%s' Value '%s'.", event.Type.String(), event.Kv.Key, event.Kv.Value)
					} else {
						description = fmt.Sprintf("%s on Key '%s'.", event.Type.String(), event.Kv.Key)
					}
					changes = append(changes, description)
				}

				ch <- changes
			}
		}
	}()

	return ch, nil
}

func (c *EtcdClient) GetKV(addKV func(string, string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := c.internalClient.Get(ctx, "", clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range response.Kvs {
		addKV(string(kv.Key), string(kv.Value))
	}

	return nil
}

func (c *EtcdClient) StoreKV(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := c.internalClient.Put(ctx, key, value)
	if err != nil {
		return err
	}

	return nil
}

func (c *EtcdClient) DeleteKey(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := c.internalClient.Delete(ctx, key)
	if err != nil {
		return err
	}

	return nil
}
