package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (c *client) sign(msg string) (string, error) {
	cmd := exec.Command(cfg.KeybasePath, "sign", "-m", msg)
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error sign: %s %s", err, outErr.String())
	}
	if outErr.String() != "" {
		log.Println(outErr.String())
	}
	if strings.HasPrefix(out.String(), "exit") {
		return "", fmt.Errorf("error signing: %s", out.String())
	}
	return out.String(), nil
}

func decrypt(rawMsg string) string {
	if !strings.HasPrefix(rawMsg, "BEGIN ") {
		rawMsg = strings.SplitN(rawMsg, " ", 2)[1]
	}
	cmd := exec.Command(cfg.KeybasePath, "decrypt", "-m", rawMsg)
	var out bytes.Buffer
	// cmd.Stdin = strings.NewReader()
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println(outErr.String())
		log.Fatal(err)
	}
	notice := outErr.String()
	notice = notice[:len(notice)-1] // remove newline
	msg := out.String()
	msg = msg[:len(msg)-1] // remove newline
	return fmt.Sprintf("%s: %s", notice, msg)
}

func decryptFile(fileName string, data io.Reader) error {
	tempPath, err := filepath.Abs("./temp/" + fileName)
	outPath, err := filepath.Abs("./files/" + fileName)
	if err != nil {
		return err
	}
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer tempFile.Close()
	io.Copy(tempFile, data)
	cmd := exec.Command(cfg.KeybasePath, "decrypt", "-i", tempPath, "-o", outPath)
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("error decrypting: %s %s", err, outErr.String())
		return err
	}
	if outErr.String() != "" {
		log.Println(outErr.String())
	}
	if strings.HasPrefix(out.String(), "exit") {
		err = fmt.Errorf("error decrypting: %s", out.String())
		return err
	}
	return err
}

func encryptFile(user, path string) (file *os.File, err error) {
	path, err = filepath.Abs(path)
	outPath, err := filepath.Abs("./temp/" + user)
	log.Println("ENCRYPT", path)
	cmd := exec.Command(cfg.KeybasePath, "encrypt", user, "-i", path, "-o", outPath)
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("error encrypting: %s %s", err, outErr.String())
		return
	}
	if outErr.String() != "" {
		log.Println(outErr.String())
	}
	if strings.HasPrefix(out.String(), "exit") {
		err = fmt.Errorf("error encrypting: %s", out.String())
		return
	}
	file, err = os.OpenFile(outPath, os.O_RDONLY, 0777)
	return file, err
}

func encrypt(user, msg string) (string, error) {
	log.Println("ENCRYPT", msg)
	cmd := exec.Command(cfg.KeybasePath, "encrypt", user, "-m", msg+"\n")
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error encrypting: %s %s", err, outErr.String())
	}
	if outErr.String() != "" {
		log.Println(outErr.String())
	}
	if strings.HasPrefix(out.String(), "exit") {
		return "", fmt.Errorf("error encrypting: %s", out.String())
	}
	return out.String(), nil
}
