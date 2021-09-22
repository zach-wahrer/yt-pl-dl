package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mikkyang/id3-go"
)

var dlArgs = []string{"--yes-playlist", "--extract-audio", "--audio-format", "mp3"}

const fsMode = 0770
const ytHashOffset = 12

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

	if err := createAndChangeDir(fmt.Sprintf("%s/%s", f.artist, f.album)); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}

	if err := download(f.url); err != nil {
		log.Printf("error running youtube-dl command: %s", err)
		os.Exit(1)
	}

	fileNames, err := getFileNames()
	if err != nil {
		log.Printf("error getting song names: %s", err)
		os.Exit(1)
	}

	fixedFileNames, err := fixFileNames(fileNames)
	if err != nil {
		log.Printf("error fixing file names: %s", err)
		os.Exit(1)
	}

	if err := tagSongs(fixedFileNames, f.artist, f.album); err != nil {
		log.Printf("error tagging mp3s: %s", err)
		os.Exit(1)
	}

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

func download(playlistUrl string) error {
	ytDlCmd := exec.Command("youtube-dl", append(dlArgs, playlistUrl)...)

	ytDlCmd.Stdout = os.Stdout
	ytDlCmd.Stderr = os.Stderr

	return ytDlCmd.Run()
}

func fixFileNames(fileNames []string) ([]string, error) {
	var fixedFileNames = []string{}
	for _, fileName := range fileNames {
		typeSplit := strings.Split(fileName, ".")
		fileType := typeSplit[len(typeSplit)-1]
		fixSplit := strings.Split(typeSplit[0], "")
		fixedFileName := strings.Join(fixSplit[0:len(fixSplit)-ytHashOffset], "")
		fixedFullFileName := fmt.Sprintf("%s.%s", fixedFileName, fileType)
		fixedFileNames = append(fixedFileNames, fixedFullFileName)

		if err := os.Rename(fileName, fixedFullFileName); err != nil {
			return nil, fmt.Errorf("error renaming file %s: %s", fileName, err)
		}
	}
	return fixedFileNames, nil
}

func getFileNames() ([]string, error) {
	files, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var fileNames = []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	return fileNames, nil
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

func tagSongs(filenames []string, artist, album string) error {
	for _, fileName := range filenames {
		if isMP3(fileName) {
			file, err := id3.Open(fileName)
			if err != nil {
				return err
			}
			defer file.Close()

			file.SetArtist(artist)
			file.SetAlbum(album)
			file.SetTitle(strings.Split(fileName, ".mp3")[0])
		}
	}

	return nil
}

func isMP3(fileName string) bool {
	fileSplit := strings.Split(fileName, ".")
	return fileSplit[len(fileSplit)-1] == "mp3"
}
