package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/JackKCWong/lineio"
)

type App struct {
}

func (a *App) Run(ctx context.Context, args []string) error {
	infile, err := os.Open(args[0])
	if err != nil {
		return err
	}

	buf := make([]byte, 4*1024)
	scanner := lineio.NewScanner(infile, buf)

	for scanner.Scan() {
		line := scanner.Line()
		fmt.Printf("%d:%d-%d\t\t%s\n", line.No, line.LineStart, line.LineEnding, line.Raw)
	}

	return scanner.Err() 
}

func main() {
	var app App
	ctx, cancel := context.WithCancel(context.Background())

	sigKill := make(chan os.Signal, 1)
	signal.Notify(sigKill, os.Interrupt)

	go func() {
		<-sigKill
		cancel()
	}()

	flag.Parse()

	if err := app.Run(ctx, flag.Args()); err != nil {
		log.Printf("exited with error: %q", err)
	}
}
