package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	//"strings"
)

type Flags struct {
	InputFile   string
	CollectCode string
	Server      *url.URL
	logLevel    slog.Level
	ChunkSize   int
}

func GetFlags() (*Flags, error) {
	flags := &Flags{}
	flag.StringVar(&flags.InputFile, "i", "", "Path to the file or folder to be sent")
	flag.StringVar(&flags.CollectCode, "c", "", "Code provided to collect a file")
	flag.IntVar(&flags.ChunkSize, "b", 16384, "Size of the chunks the file will be split into for sending in bytes")
	
	server := flag.String("s", "ws://localhost:8080/ws", "server used to relay messages")
	verbose := flag.Bool("vvv", false, "Enable verbose mode")
	flag.Parse()

	s, err := url.Parse(*server)
	if err != nil {
		return nil, fmt.Errorf("invalid server url: %w", err)
	}
	//TODO: handle difference cases of the server input, automatically adding the scheme for example
	flags.Server = s

	if *verbose {
		flags.logLevel = slog.LevelInfo
	} else {
		flags.logLevel = slog.LevelError
	}

	if flags.InputFile != "" && flags.CollectCode != "" {
		return nil, errors.New("unable to collect and accept input at the same time. ensure that only the -i or -c flag is entered")
	}

	return flags, nil
}
