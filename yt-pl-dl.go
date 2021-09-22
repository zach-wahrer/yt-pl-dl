package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
)

var dlArgs = []string{"--yes-playlist", "--extract-audio", "--audio-format", "mp3"}

type flags struct {
	url    string
	artist string
	album  string
}

func main() {
	f := getFlags()

	if f.url == "" || f.artist == "" || f.album == "" {
		printMissingFlags()
	}

	ytDlCmd := exec.Command("youtube-dl", append(dlArgs, f.url)...)

	ytDlCmd.Stdout = os.Stdout
	ytDlCmd.Stderr = os.Stderr
	if err := ytDlCmd.Run(); err != nil {
		log.Printf("error running youtube-dl command: %s", err)
		os.Exit(1)
	}

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
