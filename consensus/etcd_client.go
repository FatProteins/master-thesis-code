package consensus

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

var etcdClientLogger = daLogger.NewLogger("etcdclient")

type EtcdClient struct {
	mainClient       *clientv3.Client
	contactAllClient *clientv3.Client
}

func NewEtcdClient(ctx context.Context, mainEndpoint string, endpoints []string) *EtcdClient {
	mainClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{mainEndpoint},
		DialTimeout: 10 * time.Second,
		Context:     ctx,
	})
	if err != nil {
		panic(err)
	}

	contactAllClient, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 10 * time.Second,
		Context:     ctx,
	})
	if err != nil {
		panic(err)
	}

	return &EtcdClient{mainClient: mainClient, contactAllClient: contactAllClient}
}

func (c *EtcdClient) SubscribeToChanges(ctx context.Context) (<-chan interface{}, error) {
	ch := make(chan interface{})
	watchChan := c.contactAllClient.Watch(ctx, "", clientv3.WithPrefix())

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
	response, err := c.contactAllClient.Get(context.TODO(), "", clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range response.Kvs {
		addKV(string(kv.Key), string(kv.Value))
	}

	return nil
}

func (c *EtcdClient) StoreKV(key, value string) error {
	_, err := c.mainClient.Put(context.TODO(), key, value)
	if err != nil {
		return err
	}

	return nil
}

func (c *EtcdClient) DeleteKey(key string) error {
	_, err := c.mainClient.Delete(context.TODO(), key)
	if err != nil {
		return err
	}

	return nil
}
