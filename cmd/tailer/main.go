package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/JackKCWong/go-bulkio"
	"github.com/pkg/profile"
)

func main() {
	fProf := flag.String("prof", "mem", "cpu|mem")
	fStart := flag.String("start", "1,0", "lineno,offset")
	fBufSize := flag.Int("buf", 4, "buffer size in KBs")
	fBackoff := flag.Int("backoff", 500, "backoff time in ms")

	flag.Parse()

	switch *fProf {
	case "cpu":
		defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()
	case "mem":
		defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	default:
		fmt.Fprintf(os.Stderr, "unknown flag: %s\n", *fProf)
		fmt.Fprintln(os.Stderr, "usage: tailer -prof cpu|mem [-start lineno,offset] <file>")
		os.Exit(1)
	}

	fd, err := os.Open(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	starts := strings.Split(*fStart, ",")
	lineno, err := strconv.Atoi(starts[0])
	if err != nil {
		panic(err)
	}

	offset, err := strconv.Atoi(starts[1])
	if err != nil {
		panic(err)
	}

	tailer := bulkio.NewTailer(fd, *fBufSize*1024)
	tailer.StartingByte = int64(offset)
	tailer.StartingLine = lineno

	ctx, cancel := context.WithCancel(context.Background())

	sigKill := make(chan os.Signal, 1)
	signal.Notify(sigKill, os.Interrupt)
	go func() {
		<-sigKill
		cancel()
	}()

	err = tailer.TailN(ctx, time.Duration(*fBackoff)*time.Millisecond, func(lines []bulkio.Line) error {
		for i := range lines {
			fmt.Printf("%d:%d\t\t%s", lines[i].No, lines[i].Offset, lines[i].Raw)
		}

		fmt.Println("#########################")
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %q\n", err)
	}
}
