package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/gempir/go-twitch-irc/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

const soundsDir string = "sounds"

var (
	sounds = make(map[string]string) // key is the name, value is the path
	rate   = beep.SampleRate(44100)  // default rate for player

	// array of sound and related command available
	availableCommands = []string{
		"ding",
	}
)

func init() {
	log.SetLevel(log.TraceLevel)
}

func main() {
	if err := config(); err != nil {
		log.Fatalln(err)
	}

	if err := configureTwitch().Connect(); err != nil {
		log.Fatalf("cannot configure Twitch chat bot: %s", err)
	}
}

func config() error {
	if err := readConfigFile(); err != nil {
		return fmt.Errorf("could not read config file: %w", err)
	}

	if err := GetSounds(); err != nil {
		return fmt.Errorf("cannot read song list: %w", err)
	}

	if len(sounds) == 0 {
		return errors.New("no sound files found")
	}

	// Init the speaker
	if err := speaker.Init(rate, rate.N(time.Second/10)); err != nil {
		return fmt.Errorf("cannot init the speaker for rate [%d]: %w", rate, err)
	}

	if err := StartSound(); err != nil {
		return fmt.Errorf("could not configure speaker: %s", err)
	}

	return nil
}

func readConfigFile() error {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("toml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	return viper.ReadInConfig()
}

func configureTwitch() *twitch.Client {
	twitch.ReadBufferSize = 1                // only handle one message at once
	var client = twitch.NewAnonymousClient() // use an anonymous client, we do not need to write anything

	// register handlers
	client.OnPrivateMessage(HandleOnPrivateMessage)

	client.Join(viper.GetString("twitch_username"))
	return client
}

func isAuthorized(user string) bool {
	user = strings.ToLower(user)

	// authorize the channel owner
	if user == strings.ToLower(viper.GetString("twitch_username")) {
		return true
	}

	allAuthorizedUsers := viper.GetStringSlice("twitch_authorized_users")
	// Let all users play sound effects if we haven't specified a list of authorized users
	if len(allAuthorizedUsers) == 0 {
		return true
	}

	for _, authorizedUser := range allAuthorizedUsers {
		if user == strings.ToLower(authorizedUser) {
			return true
		}
	}

	return false
}

// StartSound play a startup sound
func StartSound() error {
	return Play(filepath.Join(soundsDir, "startup.wav"))
}

// Play the song at given path
func Play(path string) error {
	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
		err      error
	)

	// read the file
	streamer, format, err = DecodeFile(path)
	if err != nil {
		return fmt.Errorf("cannot decode file [%s]: %w", path, err)
	}

	var done = make(chan struct{}) // chan to wait for playback finished
	var s beep.Streamer = streamer // finale streamer to play

	// Reset the speak if bitrate is different
	if format.SampleRate != rate {
		log.Tracef("resampling sound: speaker rate [%d], file rate [%d]", rate, format.SampleRate)
		s = beep.Resample(4, rate, format.SampleRate, streamer)
	}

	log.Printf("playing [%s] with bitrate as [%d]", path, format.SampleRate)
	speaker.Play(beep.Seq(s, beep.Callback(func() {
		done <- struct{}{}
	})))

	<-done // wait for this song played
	return streamer.Close()
}

// DecodeFile read the selected audio file
func DecodeFile(path string) (beep.StreamSeekCloser, beep.Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}

	switch {
	case strings.HasSuffix(path, ".flac"):
		return flac.Decode(f)
	case strings.HasSuffix(path, ".wav"):
		return wav.Decode(f)
	case strings.HasSuffix(path, ".mp3"):
		return mp3.Decode(f)
	default:
		return vorbis.Decode(f)
	}
}

// GetSounds returns current available sounds
func GetSounds() error {
	return filepath.Walk(soundsDir, func(path string, info fs.FileInfo, err error) error {
		if path == soundsDir {
			return nil
		}

		if err != nil {
			return fmt.Errorf("cannot walk into %s: %w", soundsDir, err)
		}

		// store sound as name without extension
		filename := info.Name()
		sounds[strings.TrimSuffix(filename, filepath.Ext(filename))] = filepath.Join(soundsDir, filename)
		return nil
	})
}

// HandleOnPrivateMessage play sound from given incoming message
func HandleOnPrivateMessage(message twitch.PrivateMessage) {
	log.Tracef("received message [%s] from [%s]", message.Message, message.User.Name)

	// If used not authorized to play a sound
	if !isAuthorized(message.User.Name) {
		log.Tracef("user [%s] is not authorized to play a sound", message.User.Name)
		return
	}

	command := strings.TrimPrefix(strings.ToLower(message.Message), "!")
	_, ok := funk.FindString(availableCommands, func(s string) bool {
		return s == command
	})

	// unhandled command
	if !ok {
		log.Tracef("unhandled [%s]", command)
		return
	}

	// search for matching song
	v, ok := sounds[command]
	if !ok {
		log.Debugf("user [%s] wanted to play unknow sound: [%s]", message.User.Name, command)
		return
	}

	log.Printf("[%s] ask to play [%s]", message.User.Name, command)
	if err := Play(v); err != nil {
		log.Errorf("cannot play sound [%s]", v)
	}
}
