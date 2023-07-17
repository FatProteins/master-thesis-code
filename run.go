package main

import (
	"context"
	"fmt"
	"github.com/FatProteins/master-thesis-code/network"
	"github.com/FatProteins/master-thesis-code/process"
	"github.com/FatProteins/master-thesis-code/setup"
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
	configString, err := faultConfig.String()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to serialize config file '%s' to yaml: %s\n", ConfigPath, err.Error())
		os.Exit(1)
	}
	fmt.Printf("Using fault config:\n%s\n", configString)

	msgChan := make(chan network.Message, 10000)
	localAddr, err := net.ResolveUnixAddr("unixgram", faultConfig.UnixDomainSocketPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not resolve unix address: %s\n", err.Error())
		os.Exit(1)
	}

	networkLayer, err := network.NewNetworkLayer(localAddr, msgChan)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not create network layer: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Listening to unix socket on '%s'\n", localAddr.String())

	actionPicker := setup.NewActionPicker(faultConfig)
	processor := process.NewProcessor(msgChan, actionPicker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Starting application...")
	networkLayer.RunAsync(ctx)
	processor.RunAsync(ctx)

	fmt.Println("Ready.")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	_ = <-interrupt
}
