package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var fileNameRegex = regexp.MustCompile(`^[\w_\- ]+\.[\w]+$`)
var fileNameBase = regexp.MustCompile(`[\w_\- ]+\.\w+$`)

func checkFileName(name string) error {
	match := fileNameRegex.FindString(name)
	if match != name {
		return errors.New("invalid file name")
	}
	return nil
}

func cleanPath(p string) string {
	if strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'") {
		p = p[1 : len(p)-1]
	}
	return p
}

func getFileName(path string) string {
	match := fileNameBase.FindString(path)
	return match
}

func dumpFile(path string) (bs []byte, err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(path)
	file, err := os.OpenFile(path, os.O_RDONLY, 0777)
	if err != nil {
		log.Println(err)
		return
	}
	bs, err = ioutil.ReadAll(file)
	if err != nil {
		return
	}
	log.Println(string(bs))
	return
}

func openFile(path string) (file *os.File, err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(path)
	file, err = os.OpenFile(path, os.O_RDONLY, 0777)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func saveFile(name string, data io.Reader) (err error) {
	path := filepath.Join("./files", name)
	if _, err = os.Stat(path); os.IsExist(err) {
		log.Println(path, "already exists")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return
	}
	defer file.Close()
	io.Copy(file, data)
	return
}
