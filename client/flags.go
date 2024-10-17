package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	//"strings"
)

type Flags struct {
	InputFile      string
	CollectCode    string
	Server         *url.URL
	logLevel       slog.Level
	ChunkSize      int
	OutputPath     string
	OutputFileName string
}

func GetFlags() (*Flags, error) {
	flags := &Flags{}
	flag.StringVar(&flags.InputFile, "i", "", "Path to the file or folder to be sent")
	flag.StringVar(&flags.CollectCode, "c", "", "Code provided to collect a file")
	flag.IntVar(&flags.ChunkSize, "b", 16384, "Size of the chunks the file will be split into for sending in bytes")
	flag.StringVar(&flags.OutputPath, "o", "", "Output path of the received file")
	flag.StringVar(&flags.OutputFileName, "f", "", "Output file name")
	//stun server
	//output path
	//output file name
	server := flag.String("r", "ws://localhost:8080/ws", "server used to relay messages")
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

	if flags.InputFile == "" && flags.CollectCode == "" {
		return nil, errors.New("adit requires a file to send or a code to collect, --help for more information")
	}
	if flags.InputFile != "" && flags.CollectCode != "" {
		return nil, errors.New("unable to collect and accept input at the same time. ensure that only the -i or -c flag is entered")
	}

	cleanOutPath, err := ensureDirExists(flags.OutputPath)
	if err != nil {
		return nil, err
	}
	flags.OutputPath = cleanOutPath

	return flags, nil
}

func ensureDirExists(dirPath string) (string, error) {
	cleanedPath := filepath.Clean(dirPath)

	info, err := os.Stat(cleanedPath)
	if err == nil && info.IsDir() {
		return cleanedPath, nil
	}

	err = os.Mkdir(cleanedPath, os.ModePerm)

	return cleanedPath, err
}
