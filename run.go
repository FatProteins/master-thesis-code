package main

import (
	"context"
	"github.com/FatProteins/master-thesis-code/consensus"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/process"
	"github.com/FatProteins/master-thesis-code/rest"
	"github.com/FatProteins/master-thesis-code/setup"
	"net"
	"os"
	"os/signal"
	"strings"
)

var logger = daLogger.NewLogger("main")

func Run() {
	faultConfig, err := setup.ReadFaultConfig(ConfigPath)
	if err != nil {
		logger.ErrorErr(err, "Could not read fault config yaml file")
		os.Exit(1)
	}
	configString, err := faultConfig.String()
	if err != nil {
		logger.ErrorErr(err, "Failed to serialize config file '%s' to yaml", ConfigPath)
		os.Exit(1)
	}
	logger.Info("Using fault config:\n%s", configString)

	localAddr, err := net.ResolveUnixAddr("unixgram", faultConfig.UnixToDaDomainSocketPath)
	if err != nil {
		logger.ErrorErr(err, "Could not resolve unix address")
		os.Exit(1)
	}

	msgChan := make(chan network.Message, 10000)
	respChan := make(chan network.Message, 10000)
	networkLayer, err := network.NewNetworkLayer(msgChan, respChan, localAddr, faultConfig.UnixToDaDomainSocketPath)
	if err != nil {
		logger.ErrorErr(err, "Could not create network layer")
		os.Exit(1)
	}
	logger.Info("Listening to unix socket on '%s'", localAddr.String())

	actionPicker := setup.NewActionPicker(faultConfig)
	processor := process.NewProcessor(msgChan, respChan, actionPicker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("Starting application...")
	networkLayer.RunAsync(ctx)
	processor.RunAsync(ctx)

	etcdClientPort := os.Getenv("ETCD_CLIENT_CONTAINER_PORT")
	if etcdClientPort == "" {
		panic("ETCD_CLIENT_CONTAINER_PORT env variable is empty or not set")
	}

	allEtcdClientPorts := os.Getenv("ALL_ETCD_CLIENT_PORTS")
	if allEtcdClientPorts == "" {
		panic("ALL_ETCD_CLIENT_PORTS env variable is empty or not set")
	}

	var allEndpoints []string
	for _, clientPort := range strings.Split(allEtcdClientPorts, ",") {
		allEndpoints = append(allEndpoints, "host.docker.internal:"+clientPort)
	}

	mainEndpoint := "etcd:" + etcdClientPort
	go rest.EducationApi(networkLayer, actionPicker, consensus.NewEtcdClient(ctx, mainEndpoint, allEndpoints))

	logger.Info("Ready.")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	_ = <-interrupt
}
