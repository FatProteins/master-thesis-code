package main

import (
	"context"
	"fmt"
	network2 "github.com/FatProteins/master-thesis/network"
	"github.com/FatProteins/master-thesis/process"
	"github.com/FatProteins/master-thesis/setup"
	"net"
	"os"
	"os/signal"
)

func Run() {
	faultConfig, err := setup.ReadFaultConfig(ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not read fault config yaml file: %s\n", err.Error())
		os.Exit(1)
	}

	msgChan := make(chan network2.Message, 10000)
	localAddr, err := net.ResolveUnixAddr("unixgram", faultConfig.UnixDomainSocketPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not resolve unix address: %s\n", err.Error())
		os.Exit(1)
	}

	networkLayer, err := network2.NewNetworkLayer(localAddr, msgChan)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not create network layer: %s\n", err.Error())
		os.Exit(1)
	}

	actionPicker := setup.NewActionPicker(faultConfig)
	processor := process.NewProcessor(msgChan, actionPicker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	networkLayer.RunAsync(ctx)
	processor.RunAsync(ctx)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	_ = <-interrupt
}
