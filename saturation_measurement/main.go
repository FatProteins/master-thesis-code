package main

import (
	"context"
	"fmt"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/spf13/pflag"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/time/rate"
	"math"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var logger = daLogger.NewLogger("main")

type kvPair struct {
	key            string
	value          string
	success        bool
	timestampStart int64
	timestampEnd   int64
}

var leaderEps []string
var targetLeader bool

func main() {
	endpointsPtr := pflag.StringSliceP("endpoints", "e", nil, "Endpoints of etcd nodes in format 'host:port'")
	numClientsPtr := pflag.IntP("num-clients", "c", 1, "Number of clients to use concurrently")
	targetLeaderPtr := pflag.BoolP("target-leader", "l", true, "Only target the leader with requests")
	pflag.Parse()

	endpoints := *endpointsPtr
	if len(endpoints) == 0 {
		logger.Error("--endpoints is required and must not be empty")
		os.Exit(1)
	}

	numClients := uint64(*numClientsPtr)
	targetLeader = *targetLeaderPtr

	for _, endpoint := range endpoints {
		matched, err := regexp.MatchString(".+:[0-9]+", endpoint)
		if err != nil {
			panic(err)
		}

		if !matched {
			logger.Error("Endpoint must have format 'host:port', but got '%s'", endpoint)
			os.Exit(1)
		}

		logger.Info("Endpoint: %s", endpoint)
	}

	logger.Info("Number of clients: %d", numClients)
	logger.Info("Target leader: %t", targetLeader)

	storageChan := make(chan kvPair, 65536)
	storageDoneChan := make(chan struct{})
	runStorage(storageChan, storageDoneChan, numClients)

	mainCtx, mainCancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer mainCancel()

	ctx, cancel := context.WithCancel(mainCtx)
	client, err := createEtcdClient(ctx, endpoints)
	if err != nil {
		panic("failed to create etcd client")
	}

	defer client.Close()

	cc := make(chan os.Signal, 1)
	signal.Notify(cc, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-cc:
			cancel()
		case <-ctx.Done():
		}
	}()

	wg := sync.WaitGroup{}
	limit := rate.NewLimiter(rate.Limit(math.MaxInt32), 1)
	var clientNum uint64
	totalMessageCount := atomic.Uint64{}
	errorsCount := atomic.Uint64{}

	logger.Info("Deploying clients: %d", numClients)
	var count uint64
	for count = 0; count < numClients; count++ {
		go func(clientCount uint64) {
			wg.Add(1)
			defer wg.Done()
			done := ctx.Done()
			var messagesCount uint64
		loop:
			for messagesCount = totalMessageCount.Add(1); true; messagesCount = totalMessageCount.Add(1) {
				limit.Wait(context.Background())
				key := strconv.FormatUint(messagesCount, 10)

				value := key

				timestampStart := time.Now().UnixNano()

				reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
				err := doNextOp(reqCtx, client, key, value)
				reqCancel()

				timestampEnd := time.Now().UnixNano()
				success := err == nil
				if !success {
					errorsCount.Add(1)
				}

				storageChan <- kvPair{
					key:            key,
					value:          value,
					success:        success,
					timestampStart: timestampStart,
					timestampEnd:   timestampEnd,
				}

				select {
				case <-done:
					break loop
				default:
				}
			}

		}(clientNum + count)
	}

	time.Sleep(90 * time.Second)
	cancel()

	wg.Wait()

	logger.Info("Sent %d puts", totalMessageCount.Load())
	logger.Info("%d errors during puts", errorsCount.Load())

	logger.Info("Async all done")

	close(storageChan)
	<-storageDoneChan
	logger.Info("Storage done")
}

func doNextOp(ctx context.Context, client *clientv3.Client, key string, value string) error {
	resp, err := client.Put(ctx, key, value)
	if err != nil {
		if resp != nil {
			response := (etcdserverpb.PutResponse)(*resp)
			logger.ErrorErr(err, "Failed put request for key %s and value %s: %s", key, value, (&response).String())
		} else {
			logger.ErrorErr(err, "Failed put request with no response for key %s and value %s", key, value)
		}
	}
	return err
}

func createEtcdClient(ctx context.Context, endpoints []string) (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		Context:     ctx,
	})

	if targetLeader && len(leaderEps) == 0 {
		mustFindLeaderEndpoints(client)
		client.Close()
		return createEtcdClient(ctx, leaderEps)
	}

	if err != nil {
		return nil, err
	}

	return client, nil
}

func runStorage(kv <-chan kvPair, doneChan chan<- struct{}, numClients uint64) {
	go func() {
		file := createStorageFile(numClients)
		defer file.Close()

		for {
			select {
			case pair, ok := <-kv:
				if !ok {
					close(doneChan)
					return
				}

				_, err := file.WriteString(fmt.Sprintf("%s,%s,%t,%d,%d\n", pair.key, pair.value, pair.success, pair.timestampStart, pair.timestampEnd))
				if err != nil {
					panic(err)
				}
			}
		}
	}()
}

func createStorageFile(numClients uint64) *os.File {
	now := time.Now()
	filename := fmt.Sprintf("%d-kv_pairs_%s.csv", numClients, now.Format("2006-01-02T15-04-05"))
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	_, err = file.WriteString("key,value,success,timestampStart,timestampEnd\n")
	if err != nil {
		panic(err)
	}

	return file
}

func mustFindLeaderEndpoints(c *clientv3.Client) {
	resp, lerr := c.MemberList(context.TODO())
	if lerr != nil {
		panic(fmt.Sprintf("failed to get a member list: %s", lerr))
	}

	leaderID := uint64(0)
	for _, ep := range c.Endpoints() {
		if sresp, serr := c.Status(context.TODO(), ep); serr == nil {
			leaderID = sresp.Leader
			break
		}
	}

	for _, m := range resp.Members {
		if m.ID == leaderID {
			leaderEps = m.ClientURLs
			return
		}
	}

	logger.Info("Failed to find leader endpoint. Retrying...")
	mustFindLeaderEndpoints(c)
}
