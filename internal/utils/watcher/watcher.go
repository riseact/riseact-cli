package watcher

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

const N = 3

var arr [N]int = [N]int{1, 2, 3}

const foldersCount = 7

var folders = [foldersCount]string{
	"assets",
	"config",
	"layout",
	"sections",
	"snippets",
	"templates",
	"templates/supporters",
}

type FileEventAction string

const (
	FileCreate FileEventAction = "create"
	FileWrite  FileEventAction = "write"
	FileDelete FileEventAction = "delete"
	FileMove   FileEventAction = "move"
)

type FileEvent struct {
	Action   FileEventAction "json:\"action\""
	FileName string          "json:\"file_name\""
}

func (e *FileEvent) String() string {
	return fmt.Sprintf("%s: %s", e.Action, e.FileName)
}

func WatchPath(c chan FileEvent, path string) error {
	fmt.Println("Watching themes folder: " + path)

	err := validateFolder(path)

	if err != nil {
		log.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	err = watcher.Add(path)

	if err != nil {
		log.Fatal(err)
	}

	for _, folder := range folders {
		err = watcher.Add(filepath.Join(path, folder))

		if err != nil {
			log.Fatal(err)
		}
	}

	// Start listening for events.
	watch(watcher, c)

	return nil

}

func watch(w *fsnotify.Watcher, c chan FileEvent) {
	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				w.Add(event.Name)
			}
			ce := createEventFromFSNotify(event)

			c <- ce
		case err, ok := <-w.Errors:
			if !ok {
				return
			}

			log.Println("error:", err)
		}
	}
}

func validateFolder(path string) error {
	// Check if the paths exist.
	for _, folder := range folders {
		if _, err := os.Stat(path + "/" + folder); os.IsNotExist(err) {
			return errors.New("Not a valid theme folder: folder " + path + "/" + folder + " does not exist.")
		}
	}

	return nil
}

func createEventFromFSNotify(event fsnotify.Event) FileEvent {
	var action FileEventAction

	if event.Op&fsnotify.Create == fsnotify.Create {
		action = FileCreate
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		action = FileWrite
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		action = FileDelete
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		action = FileMove
	}

	return FileEvent{
		Action:   action,
		FileName: event.Name,
	}
}
