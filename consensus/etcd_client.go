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

func NewEtcdClient(ctx context.Context, endpoints []string) *EtcdClient {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 10 * time.Second,
		Context:     ctx,
	})
	if err != nil {
		panic(err)
	}

	return &EtcdClient{internalClient: client}
}

func (c *EtcdClient) GetAllKV(addKV func(string, string)) error {
	response, err := c.internalClient.Get(context.TODO(), "", clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range response.Kvs {
		addKV(string(kv.Key), string(kv.Value))
	}

	return nil
}

func (c *EtcdClient) StoreKV(key string, value string) error {
	_, err := c.internalClient.Put(context.TODO(), key, value)
	if err != nil {
		return err
	}

	return nil
}
