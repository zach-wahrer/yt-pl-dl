package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type flags struct {
	url    string
	artist string
	album  string
}

func createAndChangeDir(path string) error {
	if err := os.MkdirAll(path, fsMode); err != nil {
		return fmt.Errorf("error creating album directory: %s", err)
	}
	if err := os.Chdir(path); err != nil {
		return fmt.Errorf("error entering album directory: %s", err)
	}
	return nil
}
func getFlags() flags {
	url := flag.String("url", "", "URL of yt playlist to download")
	artist := flag.String("artist", "", "playlist artist name")
	album := flag.String("album", "", "playlist album name")

	flag.Parse()
	return flags{
		url:    *url,
		artist: *artist,
		album:  *album,
	}
}

func printMissingFlags() {
	log.Println("Missing required flags:")
	flag.PrintDefaults()
	os.Exit(1)
}

func processSn(rawSn string) (snWithHash, fixedSn string) {
	if strings.Contains(rawSn, ".m4a") {
		snWithHash = strings.TrimSuffix(rawSn, ".m4a")

	} else if strings.Contains(rawSn, ".webm") {
		snWithHash = strings.TrimSuffix(rawSn, ".webm")
	}
	snSplit := strings.Split(snWithHash, "")
	fixedSn = strings.Join(snSplit[0:len(snSplit)-ytHashOffset], "")

	return
}
