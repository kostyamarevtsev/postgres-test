package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
)

func allDirectories(directory string) ([]string, error) {
	var directories []string

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			directories = append(directories, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return directories, nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)

	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func createFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}

func syncFileContent(src, dest string) error {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dest, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

var backupPath = "_backup"

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("could not load .env file: %v", err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		cancel()
		fmt.Println("finished")
		os.Exit(0)
	}()

	setting, err := parseConfig()
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}

	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}

	defer db.db.Close()

	err = initBackup(setting.path(), backupPath)
	if err != nil {
		log.Fatalf("init backup: %v", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("init fsnotify watcher: %v", err)
	}

	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				targetFile := strings.Replace(event.Name, "~", "", -1)
				backupFile := strings.Replace(targetFile, setting.path(), backupPath, 1)

				if event.Op&fsnotify.Write == fsnotify.Write {

					go func() {
						d, _ := diff(targetFile, backupFile)
						err := db.insert(targetFile, Modify, d)
						if err != nil {
							log.Fatalf("insert log to db: %v", err)
						}
						err = syncFileContent(targetFile, backupFile)
						if err != nil {
							log.Fatalf("sync file content: %v", err)
						}
					}()

					go func() {
						out, err := setting.execCommands(ctx)
						if err != nil {
							log.Fatalf("exec commands: %v", err)
						}
						fmt.Println(out)
					}()

				}

				if event.Op&fsnotify.Create == fsnotify.Create {
					if fileExists(backupFile) {
						continue
					}

					go func() {
						fileInfo, err := os.Stat(targetFile)
						if err != nil {
							log.Fatalf("stat file %s : %v", fsnotify.Create, err)
						}

						if fileInfo.IsDir() {
							err = os.MkdirAll(backupFile, os.ModePerm)
							if err != nil {
								log.Fatalf("mkdir, op %s : %v", fsnotify.Create, err)
							}

							err = watcher.Add(targetFile)
							if err != nil {
								log.Fatalf("watcher add, op %s : %v", fsnotify.Create, err)
							}

						} else {
							err = createFile(backupFile)
							if err != nil {
								log.Fatalf("createFile, op %s : %v", fsnotify.Create, err)
							}
						}

						err = db.insert(targetFile, Create, nil)
						if err != nil {
							log.Fatalf("insert log to db: %v", err)
						}
					}()

					go func() {
						out, err := setting.execCommands(ctx)
						if err != nil {
							log.Fatalf("exec commands: %v", err)
						}
						fmt.Println(out)
					}()

				}

				if event.Op&fsnotify.Remove == fsnotify.Remove {
					if fileExists(targetFile) {
						continue
					}

					go func() {
						err = os.Remove(backupFile)
						if err != nil {
							log.Fatalf("remove file: %v", err)
							return
						}

						err = db.insert(targetFile, Remove, nil)
						if err != nil {
							log.Fatalf("insert log to db: %v", err)
						}
					}()

					go func() {
						out, err := setting.execCommands(ctx)
						if err != nil {
							log.Fatalf("exec commands: %v", err)
						}
						fmt.Println(out)
					}()

				}

				if event.Op&fsnotify.Rename == fsnotify.Rename {

					go func() {
						err = os.Remove(backupFile)
						if err != nil {
							log.Fatalf("remove file: %v", err)
							return
						}
					}()

					go func() {
						out, err := setting.execCommands(ctx)
						if err != nil {
							log.Fatalf("exec commands: %v", err)
						}
						fmt.Println(out)
					}()

				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	directories, err := allDirectories(setting.path())
	if err != nil {
		log.Fatalf("allDirectories: %v", err)
	}

	for _, dir := range directories {
		err = watcher.Add(dir)
		if err != nil {
			log.Fatalf("watcher add: %v", err)
		}
	}

	<-make(chan struct{})
}
