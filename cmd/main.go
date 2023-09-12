package main

import (
	"fmt"
	"os"
	"os/signal"
	"riseact/cmd/cli"
	"syscall"
)

func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

func main() {
	cli.Execute()
}
