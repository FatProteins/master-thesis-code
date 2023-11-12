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
var leaderId uint64
var targetLeader bool
var markerKey string
var markerValue string
var faultLeader bool
var clientPerNode = make(map[uint64]*clientv3.Client)

func main() {
	endpointsPtr := pflag.StringSliceP("endpoints", "e", nil, "Endpoints of etcd nodes in format 'host:port'")
	numClientsPtr := pflag.IntP("num-clients", "c", 1, "Number of clients to use concurrently")
	targetLeaderPtr := pflag.BoolP("target-leader", "l", true, "Only target the leader with requests")
	faultLeaderPtr := pflag.Bool("fault-leader", true, "Fault the leader")
	markerValuePtr := pflag.String("marker-value", "C", "Marker value")
	pflag.Parse()

	endpoints := *endpointsPtr
	if len(endpoints) == 0 {
		logger.Error("--endpoints is required and must not be empty")
		os.Exit(1)
	}

	numClients := uint64(*numClientsPtr)
	targetLeader = *targetLeaderPtr
	markerValue = *markerValuePtr
	faultLeader = *faultLeaderPtr

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
	logger.Info("Fault leader: %t", faultLeader)
	logger.Info("Marker value: %s", markerValue)
	time.Sleep(5 * time.Second)

	//generateRandomPayload64BaseEncoded()
	//value := string(buffer)

	asyncDone := make(chan struct{}, numClients)

	storageChan := make(chan kvPair, 65536)
	storageDoneChan := make(chan struct{})
	fileMarkerKey := "L"
	if !faultLeader {
		fileMarkerKey = "F"
	}
	runStorage(storageChan, storageDoneChan, numClients, fileMarkerKey, markerValue)

	mainCtx, mainCancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer mainCancel()

	ctx, cancel := context.WithCancel(mainCtx)
	aCtx, aCancel := context.WithCancel(ctx)
	defer aCancel()
	aClient, err := createEtcdClient(aCtx, endpoints[0:1])
	if err != nil {
		panic("failed to create etcd client")
	}

	bCtx, bCancel := context.WithCancel(ctx)
	defer bCancel()
	bClient, err := createEtcdClient(bCtx, endpoints[1:2])
	if err != nil {
		panic("failed to create etcd client")
	}

	cCtx, cCancel := context.WithCancel(ctx)
	defer cCancel()
	cClient, err := createEtcdClient(cCtx, endpoints[2:3])
	if err != nil {
		panic("failed to create etcd client")
	}

	dCtx, dCancel := context.WithCancel(ctx)
	defer dCancel()
	dClient, err := createEtcdClient(dCtx, endpoints[3:4])
	if err != nil {
		panic("failed to create etcd client")
	}

	eCtx, eCancel := context.WithCancel(ctx)
	defer eCancel()
	eClient, err := createEtcdClient(eCtx, endpoints[4:5])
	if err != nil {
		panic("failed to create etcd client")
	}

	mustFindLeaderEndpoints([]*clientv3.Client{aClient, bClient, cClient, dClient, eClient})

	clientPerNode[0x30ae2677002dc3c1] = aClient
	clientPerNode[0x4805910d3c1d5962] = bClient
	clientPerNode[0x8380732e2b2bf3e2] = cClient
	clientPerNode[0xab662f865c0696a1] = dClient
	clientPerNode[0xdfdc938d196b5c46] = eClient

	atomicClient := atomic.Value{}
	atomicClient.Store(clientPerNode[leaderId])

	leaderKey := "A"
	if leaderId == 0x30ae2677002dc3c1 {
		leaderKey = "A"
	} else if leaderId == 0x4805910d3c1d5962 {
		leaderKey = "B"
	} else if leaderId == 0x8380732e2b2bf3e2 {
		leaderKey = "C"
	} else if leaderId == 0xab662f865c0696a1 {
		leaderKey = "D"
	} else if leaderId == 0xdfdc938d196b5c46 {
		leaderKey = "E"
	} else {
		panic("Could not identify leader")
	}

	if faultLeader {
		markerKey = leaderKey
	} else {
		if leaderKey == "D" {
			markerKey = "E"
		} else {
			markerKey = "D"
		}
	}

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
	for clientNum = 0; clientNum < numClients; clientNum++ {
		//break
		go func(clientCount uint64) {
			done := ctx.Done()
			//retryFromRestart := false
			var messagesCount uint64
			sentFault := false
			researchLeader := false
			startTime := time.Now()
			var subCancel context.CancelFunc
			defer func() {
				if subCancel != nil {
					subCancel()
				}
			}()

		loop:
			for messagesCount = totalMessageCount.Add(1); true; messagesCount = totalMessageCount.Add(1) {
				limit.Wait(context.Background())
				//logger.Debug("Sending put %d", messagesCount)
				key := strconv.FormatUint(messagesCount, 10)
				if clientCount == 0 && !sentFault && time.Since(startTime) > time.Second*35 {
					sentFault = true
					key = markerKey
					researchLeader = faultLeader && markerValue == "C"
				}

				timestampStart := time.Now().UnixNano()

				reqCtx, reqCancel := context.WithTimeout(ctx, 1*time.Second)
				err := doNextOp(reqCtx, atomicClient.Load().(*clientv3.Client), key, markerValue)
				reqCancel()

				timestampEnd := time.Now().UnixNano()
				success := err == nil
				if !success {
					errorsCount.Add(1)
					//break loop
				}

				if researchLeader {
					researchLeader = false
					var clients []*clientv3.Client
					for id, c := range clientPerNode {
						if leaderId == id {
							continue
						}

						clients = append(clients, c)
					}

				retry:
					mustFindLeaderEndpoints(clients)
					var subCtx context.Context
					subCtx, subCancel = context.WithCancel(ctx)
					newClient, err := createEtcdClient(subCtx, clientPerNode[leaderId].Endpoints())
					if err != nil {
						panic(err)
					}
					clientPerNode[leaderId] = newClient
					atomicClient.Store(newClient)
					//atomicClient.Store(clientPerNode[leaderId])

					reqCtx, reqCancel = context.WithTimeout(ctx, 1*time.Second)
					err = doNextOp(reqCtx, atomicClient.Load().(*clientv3.Client), strconv.FormatUint(totalMessageCount.Add(1), 10), markerValue)
					reqCancel()
					if err != nil {
						subCancel()
						goto retry
					}
				}

				storageChan <- kvPair{
					key:            key,
					value:          markerValue,
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

	time.AfterFunc(90*time.Second, cancel)

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
		DialTimeout: 1 * time.Second,
		Context:     ctx,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

func runStorage(kv <-chan kvPair, doneChan chan<- struct{}, numClients uint64, markerKey string, markerValue string) {
	go func() {
		now := time.Now()
		filename := fmt.Sprintf("%d-%s-%s-etcd_%s.csv", numClients, markerKey, markerValue, now.Format("2006-01-02T15-04-05"))
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

func mustFindLeaderEndpoints(c []*clientv3.Client) {
	for _, client := range c {
		//logger.Info("Requesting status at node %s", client.Endpoints()[0])
		if sresp, serr := client.Status(context.TODO(), client.Endpoints()[0]); serr == nil {
			if sresp.Leader == 0 || sresp.Leader == leaderId {
				continue
			}
			leaderId = sresp.Leader
			logger.Info("Found leader at node %s with leaderId %x", client.Endpoints()[0], leaderId)
			return
		}
	}

	//logger.Info("Failed to find leader endpoint. Retrying...")
	mustFindLeaderEndpoints(c)
}
