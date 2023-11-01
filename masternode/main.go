package main

import (
	"context"
	"encoding/base64"
	"fmt"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/spf13/pflag"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/time/rate"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strconv"
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
var crashKey string
var crashLeader bool

func main() {
	endpointsPtr := pflag.StringSliceP("endpoints", "e", nil, "Endpoints of etcd nodes in format 'host:port'")
	numClientsPtr := pflag.IntP("num-clients", "c", 1, "Number of clients to use concurrently")
	numMsgsPtr := pflag.IntP("num-msgs", "m", 100000, "Number of messages to send in total")
	targetLeaderPtr := pflag.BoolP("target-leader", "l", true, "Only target the leader with requests")
	crashKeyPtr := pflag.String("crash-key", "D", "Crash key")
	crashLeaderPtr := pflag.Bool("crash-leader", false, "Crash the leader")
	pflag.Parse()

	endpoints := *endpointsPtr
	if len(endpoints) == 0 {
		logger.Error("--endpoints is required and must not be empty")
		os.Exit(1)
	}

	numClients := uint64(*numClientsPtr)
	numMsgs := uint64(*numMsgsPtr)
	targetLeader = *targetLeaderPtr
	crashKey = *crashKeyPtr
	crashLeader = *crashLeaderPtr

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
	logger.Info("Number of messages in total: %d", numMsgs)
	logger.Info("Target leader: %t", targetLeader)
	logger.Info("Crash key: %s", crashKey)
	logger.Info("Crash leader: %t", crashLeader)

	//generateRandomPayload64BaseEncoded()
	//value := string(buffer)

	asyncDone := make(chan struct{}, numClients)

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

	//_, err = client.Put(ctx, "testing5", "test_value")
	//if err != nil {
	//	panic(err)
	//}
	//logger.Info("Success on putting test_key!")
	//time.Sleep(60 * time.Second)
	limit := rate.NewLimiter(rate.Limit(math.MaxInt32), 1)
	var clientNum uint64
	totalMessageCount := atomic.Uint64{}
	errorsCount := atomic.Uint64{}
	startTime := time.Now()
	for clientNum = 0; clientNum < numClients; clientNum++ {
		//break
		go func(clientCount uint64) {
			done := ctx.Done()
			//retryFromRestart := false
			var messagesCount uint64
			sentCrash := false
			researchLeader := false
		loop:
			for messagesCount = totalMessageCount.Add(1); totalMessageCount.Load() < numMsgs; messagesCount = totalMessageCount.Add(1) {
				limit.Wait(context.Background())
				//logger.Debug("Sending put %d", messagesCount)
				key := strconv.FormatUint(messagesCount, 10)
				if clientCount == 0 && !sentCrash && time.Since(startTime) > time.Second*35 {
					sentCrash = true
					key = crashKey
					researchLeader = crashLeader
				}

				//generateRandomPayload64BaseEncoded()
				//value = string(buffer)
				value := key

				//if messagesCount % 10000 == 0 {
				//	logger.Info("Sending key %s and value %s", key, value)
				//}

				//if key == "10101" {
				//	logger.Info("Waiting 5 seconds for leader to restart")
				//	time.Sleep(5 * time.Second)
				//	logger.Info("Sending key %s and value %s", key, value)
				//}

				//if messagesCount == 11000 {
				//if !retryFromRestart {
				//	messagesCount = 10101
				//	retryFromRestart = true
				//	logger.Info("Resending from key 10100")
				//	continue loop
				//}

				//	break loop
				//}

				//if retryFromRestart && messagesCount >= 10101 && messagesCount <= 10104 {
				//	reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
				//	logger.Info("Client get request BEFORE PUT for key %s", key)
				//	resp, err := client.Get(reqCtx, key)
				//	if err != nil {
				//		if resp != nil {
				//			response := (etcdserverpb.RangeResponse)(*resp)
				//			logger.ErrorErr(err, "Failed get request for key %s and value %s: %s", key, value, (&response).String())
				//		} else {
				//			logger.ErrorErr(err, "Failed get request with no response for key %s and value %s", key, value)
				//		}
				//	} else {
				//		for _, kv := range resp.Kvs {
				//			logger.Info("Got response key %s and value %s", kv.Key, kv.Value)
				//		}
				//	}
				//	reqCancel()
				//}

				timestampStart := time.Now().UnixNano()

				reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
				err := doNextOp(reqCtx, client, key, value)
				reqCancel()

				timestampEnd := time.Now().UnixNano()
				success := err == nil
				if !success {
					errorsCount.Add(1)
					//break loop
				}

				//if retryFromRestart && messagesCount >= 10101 && messagesCount <= 10104 {
				//	reqCtx, reqCancel = context.WithTimeout(ctx, 5*time.Second)
				//	logger.Info("Client get request AFTER PUT for key %s", key)
				//	resp, err := client.Get(reqCtx, key)
				//	if err != nil {
				//		if resp != nil {
				//			response := (etcdserverpb.RangeResponse)(*resp)
				//			logger.ErrorErr(err, "Failed get request for key %s and value %s: %s", key, value, (&response).String())
				//		} else {
				//			logger.ErrorErr(err, "Failed get request with no response for key %s and value %s", key, value)
				//		}
				//	} else {
				//		for _, kv := range resp.Kvs {
				//			logger.Info("Got response key %s and value %s", kv.Key, kv.Value)
				//		}
				//	}
				//	reqCancel()
				//}

				storageChan <- kvPair{
					key:            key,
					value:          value,
					success:        success,
					timestampStart: timestampStart,
					timestampEnd:   timestampEnd,
				}

				if researchLeader {
					researchLeader = false
					leaderEps = nil
					client.Close()
					client, err = createEtcdClient(ctx, endpoints)
					if err != nil {
						panic(err)
					}
				}

				select {
				case <-done:
					break loop
				default:
				}
			}

			asyncDone <- struct{}{}
		}(clientNum)
	}

	cc := make(chan os.Signal, 1)
	signal.Notify(cc, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-cc:
			cancel()
		case <-ctx.Done():
		}
	}()

	time.AfterFunc(180*time.Second, cancel)

	var i uint64
	for i = 0; i < numClients; i++ {
		//break
		_ = <-asyncDone
	}

	logger.Info("Sent %d puts", totalMessageCount.Load())
	logger.Info("%d errors during puts", errorsCount.Load())

	cancel()

	logger.Info("Async all done")

	close(storageChan)
	<-storageDoneChan
	logger.Info("Storage done")

	//rd, err := client.Snapshot(mainCtx)
	//if err != nil {
	//	panic(err)
	//}
	//defer rd.Close()

	//for _, endpoint := range endpoints {
	//	clientCtx, clientCancel := context.WithTimeout(ctx, 10*time.Second)
	//	getClient, err := createEtcdClient(clientCtx, []string{endpoint})
	//	if err != nil {
	//		logger.ErrorErr(err, "Failed to create client for GET KVs for endpoint %s", endpoint)
	//		clientCancel()
	//		continue
	//	}
	//
	//	resp, err := getClient.Get(clientCtx, "", clientv3.WithPrefix())
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	clientCancel()
	//
	//	now := time.Now()
	//	nowStr := now.Format("2006-01-02T15-04-05")
	//	f, err := os.OpenFile(fmt.Sprintf("snapshot_%s_%s", nowStr, endpoint), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	f.WriteString("key,value\n")
	//	for _, kv := range resp.Kvs {
	//		f.WriteString(string(kv.Key) + "," + string(kv.Value) + "\n")
	//	}
	//
	//	f.Close()
	//}

	//now := time.Now()
	//nowStr := now.Format("2006-01-02T15-04-05")
	//f, err := os.OpenFile(fmt.Sprintf("snapshot_%s", nowStr), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	//if err != nil {
	//	panic(err)
	//}
	//defer f.Close()

	//size, err := io.Copy(os.Stdout, rd)
	//size, err := io.Copy(f, rd)
	//if err != nil {
	//	panic(err)
	//}

	//logger.Info("Wrote snapshot of %d bytes.", size)
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

const payloadLength = 8

var seed int64 = 111
var random = rand.New(rand.NewSource(seed))
var randomBuffer = make([]byte, payloadLength, payloadLength)
var buffer = make([]byte, base64.StdEncoding.EncodedLen(payloadLength), base64.StdEncoding.EncodedLen(payloadLength))

func generateRandomPayload64BaseEncoded() []byte {
	_, _ = random.Read(randomBuffer)
	base64.StdEncoding.Encode(buffer, randomBuffer)
	return buffer
}

func createEtcdClient(ctx context.Context, endpoints []string) (*clientv3.Client, error) {
	//endpoints := make([]string, numNodes)
	//for i := 0; i < numNodes; i++ {
	//	clientPort := 2379 + i
	//	endpoints[i] = "localhost:" + strconv.Itoa(clientPort)
	//}
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
		now := time.Now()
		filename := fmt.Sprintf("%d-kv_pairs_%s.csv", numClients, now.Format("2006-01-02T15-04-05"))
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}

		defer file.Close()

		_, err = file.WriteString("key,value,success,timestampStart,timestampEnd\n")
		if err != nil {
			panic(err)
		}

		for {
			select {
			case pair, ok := <-kv:
				if !ok {
					close(doneChan)
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
