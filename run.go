package main

import (
	"context"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/process"
	"github.com/FatProteins/master-thesis-code/setup"
	"net"
	"os"
	"os/signal"
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

	localAddr, err := net.ResolveUnixAddr("unixgram", faultConfig.UnixDomainSocketPath)
	if err != nil {
		logger.ErrorErr(err, "Could not resolve unix address")
		os.Exit(1)
	}

	msgChan := make(chan network.Message, 10000)
	respChan := make(chan network.Message, 10000)
	networkLayer, err := network.NewNetworkLayer(localAddr, msgChan, respChan)
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

	logger.Info("Ready.")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	_ = <-interrupt
}
