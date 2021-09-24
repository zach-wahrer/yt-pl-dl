package main

import (
	"bufio"
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

var dlArgs = []string{"--yes-playlist", "--extract-audio", "--audio-format", "mp3", "--audio-quality", "0"}

const fsMode = 0770
const ytHashOffset = 12

type (
	songList struct {
		sync.RWMutex
		tracks map[songName]trackNumber
	}

	songName    string
	trackNumber int

	fileConversion struct {
		snWithHash string
		fixedSn    string
		artist     string
		album      string
		sl         *songList
	}
)

func main() {
	f := getFlags()

	if f.url == "" || f.artist == "" || f.album == "" {
		printMissingFlags()
	}

	if err := createAndChangeDir(fmt.Sprintf("%s/%s", f.artist, f.album)); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}

	downloadAndConvertPlaylist(f.url, f.artist, f.album)
}

func downloadAndConvertPlaylist(playlistUrl, artist, album string) {
	wg := new(sync.WaitGroup)

	ytDlCmd := exec.Command("youtube-dl", append(dlArgs, playlistUrl)...)
	ytDlCmd.Stderr = os.Stderr
	stdout, err := ytDlCmd.StdoutPipe()
	if err != nil {
		log.Printf("error setting up output: %s", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(stdout)
	go manageConversions(wg, reader, artist, album)

	if err := ytDlCmd.Run(); err != nil {
		log.Printf("error running youtube-dl command: %s", err)
	}

	wg.Wait()
}

func manageConversions(wg *sync.WaitGroup, reader io.Reader, artist, album string) {
	sl := new(songList)
	tn := trackNumber(1)
	sl.tracks = make(map[songName]trackNumber)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "[download] Downloading playlist:") {
			fmt.Printf("%s\n", strings.TrimPrefix(scanner.Text(), "[download] "))
		} else if strings.Contains(scanner.Text(), "[download] Destination:") {
			rawSn := strings.TrimPrefix(scanner.Text(), "[download] Destination: ")
			snWithHash, fixedSn := processSn(rawSn)

			fmt.Printf("Downloading track %d: %s\n", tn, fixedSn)

			sl.Lock()
			sl.tracks[songName(fixedSn)] = tn
			sl.Unlock()
			tn++

			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "Deleting original file") {
					wg.Add(1)
					fc := fileConversion{
						snWithHash: snWithHash,
						fixedSn:    fixedSn,
						artist:     artist,
						album:      album,
						sl:         sl,
					}

					go fixFileNameAndTagMP3(wg, fc)
					break
				}
			}
		} else if strings.Contains(scanner.Text(), "[download] Finished downloading playlist:") {
			break
		}
	}
}

func fixFileNameAndTagMP3(wg *sync.WaitGroup, fc fileConversion) {
	defer wg.Done()
	fullFixedFn := fmt.Sprintf("%s.mp3", fc.fixedSn)
	snWithHashAndPostfix := fmt.Sprintf("%s.mp3", fc.snWithHash)

	if err := os.Rename(snWithHashAndPostfix, fullFixedFn); err != nil {
		fmt.Printf("error renaming file %s: %s\n", snWithHashAndPostfix, err)
		return
	}
	if err := tagSong(fc); err != nil {
		fmt.Printf("error tagging mp3 %s: %s\n", fc.fixedSn, err)
	}
}

func tagSong(fc fileConversion) error {
	fmt.Printf("Tagging MP3: %s\n", fc.fixedSn)
	fn := fmt.Sprintf("%s.mp3", fc.fixedSn)

	file, err := id3.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	file.SetArtist(fc.artist)
	file.SetAlbum(fc.album)
	file.SetTitle(fc.fixedSn)
	
	fc.sl.RLock()
	defer fc.sl.RUnlock()
	if number, found := fc.sl.tracks[songName(fc.fixedSn)]; found {
		textFrame := v2.NewTextFrame(v2.V23FrameTypeMap["TRCK"], fmt.Sprintf("%d", number))
		file.AddFrames(textFrame)
	}

	return nil
}
