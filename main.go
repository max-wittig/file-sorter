package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/urfave/cli"
)

type simpleFile struct {
	fileName     string
	sortCriteria string
	directory    string
}

func (f *simpleFile) OldPath() string {
	return filepath.Join(f.directory, f.fileName)
}

func (f *simpleFile) NewPath() string {
	return filepath.Join(f.directory, f.sortCriteria, f.fileName)
}

func createFolders(path *string, fileMap *map[*simpleFile]string) {
	for _, criteria := range *fileMap {
		err := os.MkdirAll(filepath.Join(*path, criteria), 0755)
		if err != nil {
			log.Fatalln("Could not create folders")
		}
	}
}

func writeIgnoreFile(path *string, ignoreFileName string, ignoredFiles *[]string) {
	var ignoredFilesBuffer bytes.Buffer
	for _, fileName := range *ignoredFiles {
		ignoredFilesBuffer.WriteString(fileName)
		ignoredFilesBuffer.WriteString("\n")
	}

	f, err := os.Create(filepath.Join(*path, ignoreFileName))
	defer f.Close()

	if err != nil {
		log.Panicln("could not create ignore file")
	}
	f.WriteString(ignoredFilesBuffer.String())
}

func directoryHash(directory *string) string {
	dirHash := md5.New()
	err := filepath.Walk(*directory, func(pathName string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		dirFile, _ := os.Open(pathName)
		defer dirFile.Close()

		io.Copy(dirHash, dirFile)
		return nil
	})
	if err != nil {
		log.Fatalln("could not hash directory")
	}
	return hex.EncodeToString(dirHash.Sum(nil))
}

func fileHash(path *string) string {
	file, err := os.Open(*path)
	defer file.Close()

	fileMd5 := md5.New()
	fileInfo, _ := file.Stat()

	if fileInfo.IsDir() {
		io.WriteString(fileMd5, directoryHash(path))
	} else {
		_, err = io.Copy(fileMd5, file)
	}

	if err != nil {
		log.Panicln(err)
	}

	return hex.EncodeToString(fileMd5.Sum(nil))
}

func moveFiles(path *string, fileMap *map[*simpleFile]string) {
	var wg sync.WaitGroup
	wg.Add(len(*fileMap))
	for file := range *fileMap {
		go func(file *simpleFile) {
			defer wg.Done()
			oldName := file.OldPath()
			newName := file.NewPath()
			if _, err := os.Stat(newName); err != nil {
				if os.IsNotExist(err) {
					err = os.Rename(oldName, newName)
					if err != nil {
						log.Fatalln("could not rename file")
					}
				}
			} else {
				//compare hashes
				oldFileHash := fileHash(&oldName)
				newFileHash := fileHash(&newName)

				if oldFileHash == newFileHash {
					//same file. Delete source
					err = os.RemoveAll(oldName)
					if err != nil {
						log.Fatalln("could not remove file")
					}
				} else {
					// same name. Different files/folders
					newName = fmt.Sprintf("%s-%s.%s",
						filepath.Join(*path, file.sortCriteria, newName),
						oldFileHash,
						file.sortCriteria,
					)

					// try renaming. Delete source if name is still equal
					if _, err := os.Stat(newName); err != nil {
						if os.IsNotExist(err) {
							os.Rename(oldName, newName)
						}
					} else {
						err = os.RemoveAll(oldName)
						if err != nil {
							log.Fatalln("could not remove duplicate files")
						}
					}
				}
			}
		}(file)
	}

	wg.Wait()
}

func sortFiles(path *string, fileMap *map[*simpleFile]string) {
	createFolders(path, fileMap)
	moveFiles(path, fileMap)
}

func parseIgnoredFiles(path *string, fileName string) []string {
	f, err := os.Open(filepath.Join(*path, fileName))
	defer f.Close()
	if err != nil {
		if os.IsNotExist(err) {
			return make([]string, 0)
		}

		log.Panicln(err)
	}

	var ignoredFiles []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		//check if .file-sorter file is up-to-date
		filePath := filepath.Join(*path, scanner.Text())
		if _, err := os.Stat(filePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}
		ignoredFiles = append(ignoredFiles, scanner.Text())
	}
	return ignoredFiles
}

func isIgnoredFile(fileName string, ignoredFiles *[]string) bool {
	for _, file := range *ignoredFiles {
		if file == fileName {
			return true
		}
	}
	return false
}

func addToIgnoredFiles(toAdd *string, ignoredFiles *[]string) {
	for _, file := range *ignoredFiles {
		if file == *toAdd {
			return
		}
	}
	*ignoredFiles = append(*ignoredFiles, *toAdd)
}

func getFileMap(files *[]os.FileInfo, ignoredDirs *[]string, ignoreFileName string, directory *string, sortCriteria string) (*map[*simpleFile]string, error) {
	// ignore .file-sorter and binary itself, to enable ./file-sorter .
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Could not get cwd")
	}
	executablePath := filepath.Join(cwd, os.Args[0])
	fileMap := make(map[*simpleFile]string)
	for _, file := range *files {
		fileName := file.Name()
		fileExtension := strings.ToLower(strings.TrimLeft(filepath.Ext(file.Name()), "."))
		fileModificationDate := file.ModTime().Format("2006-01-02")
		if file.IsDir() {
			if isIgnoredFile(file.Name(), ignoredDirs) {
				continue
			}

			fileExtension = "dirs"
		}
		if fileExtension == "" {
			fileExtension = "none"
		}

		if fileName == ignoreFileName || executablePath == filepath.Join(*directory, fileName) {
			continue
		}

		file := simpleFile{
			sortCriteria: fileExtension,
			directory:    *directory,
			fileName:     fileName,
		}
		if sortCriteria == "mod" {
			file.sortCriteria = fileModificationDate
		} else if sortCriteria != "ext" {
			return nil, errors.New("sort order needs to be 'mod' or 'ext'")
		}

		addToIgnoredFiles(&file.sortCriteria, ignoredDirs)
		fileMap[&file] = file.sortCriteria
	}
	return &fileMap, nil
}

func fileSorter(sortCriteria string, directory string) {
	const ignoreFileName = ".file-sorter"
	if sortCriteria != "ext" && sortCriteria != "mod" {
		log.Fatalln("Please specify a valid sort criteria using the -c flag. (Can be 'mod' or 'ext')")
	}

	files, err := ioutil.ReadDir(directory)
	if err != nil {
		log.Fatalln("could not read directory")
	}

	ignoredDirs := parseIgnoredFiles(&directory, ignoreFileName)
	fileMap, err := getFileMap(&files, &ignoredDirs, ignoreFileName, &directory, sortCriteria)
	if err != nil {
		log.Fatalln(err)
	}
	sortFiles(&directory, fileMap)
	writeIgnoreFile(&directory, ignoreFileName, &ignoredDirs)
	log.Printf("Sorted %s\n", directory)
}

func main() {
	var directory string
	var sortCriteria string

	app := cli.NewApp()
	app.Name = "file-sorter"
	app.Version = "0.0.3"
	app.Usage = "Sorts files into directories, based on their file extension or modification date"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "c, criteria",
			Value:       "ext",
			Usage:       "Sort criteria of the files (ext|mod). Default: ext",
			Destination: &sortCriteria,
		},
	}

	app.Action = func(c *cli.Context) error {
		if c.NArg() > 0 {
			directory = c.Args().Get(0)
			if _, err := os.Stat(directory); os.IsNotExist(err) {
				log.Fatalln("directory does not exist")
			}
			fileSorter(sortCriteria, directory)
		} else {
			cli.ShowAppHelpAndExit(c, 1)
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Panicln(err)
	}
}
