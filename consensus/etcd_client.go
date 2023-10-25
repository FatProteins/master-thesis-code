package consensus

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

var etcdClientLogger = daLogger.NewLogger("etcdclient")

type EtcdClient struct {
	internalClient *clientv3.Client
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

	return &EtcdClient{internalClient: mainClient}
}

func (c *EtcdClient) SubscribeToChanges(ctx context.Context) (<-chan interface{}, error) {
	ch := make(chan interface{})
	watchChan := c.internalClient.Watch(ctx, "", clientv3.WithPrefix())

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-watchChan:
				if !ok {
					close(ch)
					return
				}

				ch <- struct{}{}
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
