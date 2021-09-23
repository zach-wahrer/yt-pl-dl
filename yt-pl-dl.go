package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/mikkyang/id3-go"
	v2 "github.com/mikkyang/id3-go/v2"
)

var dlArgs = []string{"--yes-playlist", "--extract-audio", "--audio-format", "mp3"}

const fsMode = 0770
const ytHashOffset = 12

type trackListing struct {
	mu      sync.Mutex
	listing map[string]int
}
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

	trackList, err := download(f.url)
	if err != nil {
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

	if err := tagSongs(fixedFileNames, trackList, f.artist, f.album); err != nil {
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

func createAndDisplayStatus(reader io.Reader, trackList *trackListing) {
	trackPosition := 1
	trackList.listing = make(map[string]int)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "[download] Downloading playlist:") {
			fmt.Printf("%s\n", strings.TrimPrefix(scanner.Text(), "[download] "))
		} else if strings.Contains(scanner.Text(), "[download] Destination:") {
			track := strings.TrimPrefix(scanner.Text(), "[download] Destination: ")

			trackWithHash := ""
			if strings.Contains(track, ".m4a") {
				trackWithHash = strings.TrimSuffix(track, ".m4a")

			} else if strings.Contains(track, ".webm") {
				trackWithHash = strings.TrimSuffix(track, ".webm")
			}
			trackSplit := strings.Split(trackWithHash, "")
			fixedTrackName := strings.Join(trackSplit[0:len(trackSplit)-ytHashOffset], "")
			fmt.Printf("Downloading track %d: %s\n", trackPosition, fixedTrackName)

			trackList.mu.Lock()
			trackList.listing[fixedTrackName] = trackPosition
			trackList.mu.Unlock()
			trackPosition++
		} else if strings.Contains(scanner.Text(), "[download] Finished downloading playlist:") {
			break
		}
	}
}

func download(playlistUrl string) (*trackListing, error) {
	ytDlCmd := exec.Command("youtube-dl", append(dlArgs, playlistUrl)...)

	ytDlCmd.Stderr = os.Stderr
	stdout, err := ytDlCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(stdout)

	trackList := new(trackListing)
	go createAndDisplayStatus(reader, trackList)

	return trackList, ytDlCmd.Run()
}

func fixFileNames(fileNames []string) ([]string, error) {
	var fixedFileNames = []string{}
	for _, fileName := range fileNames {
		fileWithHash := strings.TrimSuffix(fileName, ".mp3")
		fileSplit := strings.Split(fileWithHash, "")
		fixedFileName := strings.Join(fileSplit[0:len(fileSplit)-ytHashOffset], "")

		fullFixedFileName := fmt.Sprintf("%s.mp3", fixedFileName)
		fixedFileNames = append(fixedFileNames, fullFixedFileName)

		if err := os.Rename(fileName, fullFixedFileName); err != nil {
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

func tagSongs(filenames []string, trackList *trackListing, artist, album string) error {
	fmt.Printf("Tagging MP3s...\n")
	for _, fileName := range filenames {
		if isMP3(fileName) {
			file, err := id3.Open(fileName)
			if err != nil {
				return err
			}
			defer file.Close()

			trackName := strings.Split(fileName, ".mp3")[0]

			file.SetArtist(artist)
			file.SetAlbum(album)
			file.SetTitle(trackName)
			if position, found := trackList.listing[trackName]; found {
				textFrame := v2.NewTextFrame(v2.V23FrameTypeMap["TRCK"], fmt.Sprintf("%d", position))
				file.AddFrames(textFrame)
			}
		}
	}

	return nil
}

func isMP3(fileName string) bool {
	fileSplit := strings.Split(fileName, ".")
	return fileSplit[len(fileSplit)-1] == "mp3"
}
