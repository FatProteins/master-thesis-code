package main

import (
	"context"
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

func main() {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379", "localhost:2380"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic("failed to create etcd client")
	}

	defer client.Close()

	var seed int64 = 111
	random := rand.New(rand.NewSource(seed))
	payload := make([]byte, 4096, 4096)
	random.Read(payload)
	payload = []byte("cool value")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	asyncDone := make(chan struct{}, 1)

	go func() {
		messagesCount := 0
		errorsCount := 0
		done := ctx.Done()
	loop:
		for {
			logger.Debug("Sending put %d", messagesCount)
			_, err := client.Put(ctx, strconv.Itoa(messagesCount), string(payload))
			if err != nil {
				errorsCount++
			}
			messagesCount++

			break loop

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

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	_ = <-interrupt
	cancel()

	_ = <-asyncDone
	rd, err := client.Snapshot(context.Background())
	if err != nil {
		panic(err)
	}
	defer rd.Close()

	resp, err := client.Get(context.Background(), "", clientv3.WithPrefix())
	if err != nil {
		panic(err)
	}

	for _, kv := range resp.Kvs {
		fmt.Println(string(kv.Key) + " " + string(kv.Value) + "\n")
	}

	fmt.Printf("More: %t\n", resp.More)
	os.Exit(0)
	now := time.Now()
	nowStr := now.Format("2006-01-02T15-04-05")
	f, err := os.OpenFile(fmt.Sprintf("snapshot_%s", nowStr), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	size, err := io.Copy(os.Stdout, rd)
	//size, err := io.Copy(f, rd)
	if err != nil {
		panic(err)
	}

	logger.Info("Wrote snapshot of %d bytes.", size)
}
