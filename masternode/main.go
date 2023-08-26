package main

import (
	"context"
	"encoding/base64"
	"fmt"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var logger = daLogger.NewLogger("main")

const numClients = 1

type kvPair struct {
	key            string
	value          string
	success        bool
	timestampStart int64
	timestampEnd   int64
}

func main() {
	generateRandomPayload64BaseEncoded()
	value := string(buffer)

	ctx, cancel := context.WithCancel(context.Background())
	asyncDone := make(chan struct{}, numClients)

	storageChan := make(chan kvPair, 65536)
	runStorage(storageChan)

	for i := 0; i < numClients; i++ {
		go func() {
			client, err := createEtcdClient(ctx)
			if err != nil {
				panic("failed to create etcd client")
			}

			defer client.Close()

			messagesCount := 0
			errorsCount := 0
			done := ctx.Done()
		loop:
			for {
				logger.Debug("Sending put %d", messagesCount)
				key := strconv.Itoa(messagesCount)
				timestampStart := time.Now().UnixNano()
				err := doNextOp(ctx, client, key, value)
				timestampEnd := time.Now().UnixNano()
				success := err == nil
				if !success {
					errorsCount++
				}

				storageChan <- kvPair{
					key:            key,
					value:          value,
					success:        success,
					timestampStart: timestampStart,
					timestampEnd:   timestampEnd,
				}
				messagesCount++

				select {
				case <-done:
					break loop
				default:
				}
			}

			logger.Info("Sent %d puts", messagesCount)
			logger.Info("%d errors during puts", errorsCount)
			asyncDone <- struct{}{}
		}()
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	_ = <-interrupt
	cancel()

	for i := 0; i < numClients; i++ {
		_ = <-asyncDone
	}

	ctx, cancel = context.WithCancel(context.Background())
	client, err := createEtcdClient(ctx)
	if err != nil {
		panic("failed to create etcd client")
	}

	defer client.Close()

	rd, err := client.Snapshot(context.Background())
	if err != nil {
		panic(err)
	}
	defer rd.Close()

	//resp, err := client.Get(context.Background(), "", clientv3.WithPrefix())
	//if err != nil {
	//	panic(err)
	//}
	//
	//for _, kv := range resp.Kvs {
	//	fmt.Println(string(kv.Key) + " " + string(kv.Value) + "\n")
	//}

	now := time.Now()
	nowStr := now.Format("2006-01-02T15-04-05")
	f, err := os.OpenFile(fmt.Sprintf("snapshot_%s", nowStr), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	//size, err := io.Copy(os.Stdout, rd)
	size, err := io.Copy(f, rd)
	if err != nil {
		panic(err)
	}

	logger.Info("Wrote snapshot of %d bytes.", size)
}

func doNextOp(ctx context.Context, client *clientv3.Client, key string, value string) error {
	_, err := client.Put(ctx, key, value)
	return err
}

const payloadLength = 4096

var seed int64 = 111
var random = rand.New(rand.NewSource(seed))
var randomBuffer = make([]byte, payloadLength, payloadLength)
var buffer = make([]byte, base64.StdEncoding.EncodedLen(payloadLength), base64.StdEncoding.EncodedLen(payloadLength))

func generateRandomPayload64BaseEncoded() []byte {
	_, _ = random.Read(randomBuffer)
	base64.StdEncoding.Encode(buffer, randomBuffer)
	return buffer
}

func createEtcdClient(ctx context.Context) (*clientv3.Client, error) {
	endpoints := make([]string, numClients)
	for i := 0; i < numClients; i++ {
		clientPort := 2379 + i
		endpoints[i] = "localhost:" + strconv.Itoa(clientPort)
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		Context:     ctx,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

func runStorage(kv <-chan kvPair) {
	go func() {
		now := time.Now()
		filename := fmt.Sprintf("kv_pairs_%s.csv", now.Format("2006-01-02T15-04-05"))
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}

		defer file.Close()

		_, err = file.WriteString("key,value,success,timestampEnd\n")
		if err != nil {
			panic(err)
		}

		for {
			select {
			case pair, ok := <-kv:
				if !ok {
					return
				}

				_, err = file.WriteString(fmt.Sprintf("%s,%s,%t,%d,%d\n", pair.key, pair.value, pair.success, pair.timestampStart, pair.timestampEnd))
				if err != nil {
					panic(err)
				}
			}
		}
	}()
}
