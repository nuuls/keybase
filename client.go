package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// xD
const (
	NOTICE  = "NOTICE"
	LOGIN   = "LOGIN"
	MESSAGE = "MESSAGE"
	VERIFY  = "VERIFY"
	KEY     = "KEY"
	FILE    = "FILE"
)

const endl byte = '\u0003'

var inputCmd = regexp.MustCompile(`\/[a-zA-z]+`)

// type payload struct {
// 	Key      string `json:"key"`
// 	Command  string `json:"command"`
// 	User     string `json:"user"`
// 	Data     []byte `json:"data"`
// 	Filename string `json:"file_name"`
// }

type client struct {
	httpClient *http.Client
	conn       net.Conn
	user       string
	key        chan string
}

func (c *client) sendRaw(s string) {
	c.write([]byte(s))

}

func (c *client) write(bs []byte) {
	bs = append(bs, endl)
	c.conn.Write(bs)
}

func (c *client) send(command, s string) {
	c.sendRaw(fmt.Sprintf("%s %s", command, s))
}

func (c *client) sendEncrypt(command, user, msg string) {
	go func() {
		str, err := encrypt(user, msg)
		if err != nil {
			log.Println(err)
		} else {
			c.sendRaw(fmt.Sprintf("%s %s %s", command, user, str))
		}
	}()
}

func (c *client) connect() {
	conn, err := net.Dial("tcp", cfg.TCPHost)
	if err != nil {
		log.Println(err)
		time.Sleep(2 * time.Second)
		c.connect()
		return
	}
	c.conn = conn
	str, err := c.sign("login")
	if err != nil {
		log.Fatal(err)
	}
	c.send(LOGIN, str)
	go c.read()
}

func (c *client) queueMessage(user, msg string) {
	log.Println("MSG", msg)
	data, err := encrypt(user, msg)
	if err != nil {
		log.Println(err)
		return
	}
	header := http.Header{}
	header.Add("targetuser", user)
	header.Add("command", MESSAGE)
	c.queue(strings.NewReader(data), header)
}

func (c *client) queue(reader io.Reader, header http.Header) {
	c.key = make(chan string)
	c.sendRaw("KEY key")
	ticker := time.NewTicker(time.Minute)
	var k string
	var ok bool
	select {
	case k, ok = <-c.key:
		log.Println("got key", k)
		close(c.key)
	case <-ticker.C:
		close(c.key)
		ticker.Stop()
	}
	if !ok {
		log.Println("server didnt respond in time")
		return
	}
	c.key = nil
	req, err := http.NewRequest(http.MethodPost, cfg.HTTPHost+"/keybase", reader)
	if err != nil {
		log.Fatal(err)
	}
	header.Add("key", k)
	req.Header = header
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	r, _ := ioutil.ReadAll(res.Body)
	log.Println(string(r))
}

func (c *client) handleMessage(line string) {
	spl := strings.SplitN(line, " ", 2)
	if len(spl) < 2 {
		log.Println("invalid message", line)
		return
	}
	msg := spl[1]
	switch spl[0] {
	case MESSAGE:
		log.Println(decrypt(msg))
	case VERIFY:
		c.sendEncrypt(VERIFY, "nuuls", decrypt(msg))
	case KEY:
		if c.key == nil {
			log.Println("unexpected key", msg)
			return
		}
		c.key <- msg
	case FILE:
		c.startFileSave(spl[1])
	case NOTICE:
		log.Println(NOTICE, msg)
	}
}

func (c *client) startFileSave(cmd string) {
	spl := strings.SplitN(cmd, " ", 3)
	if len(spl) < 3 {
		log.Println("invalid command", cmd)
		return
	}
	key := spl[1]
	log.Println("KEY", key)
	fileName := spl[2]
	if err := checkFileName(fileName); err != nil {
		log.Println(spl[0], "is trying to save", fileName, "not downloading...")
		return
	}

	body := bytes.NewReader(nil)
	req, err := http.NewRequest(http.MethodGet, cfg.HTTPHost+"/keybase", body)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("key", key)
	req.Header.Add("filename", fileName)
	req.Header.Add("command", FILE)
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Println(err)
	}
	decryptFile(fileName, res.Body)
	//saveFile(fileName, res.Body)
}

func (c *client) upload(key, fileName string, file io.Reader) {
	req, err := http.NewRequest(http.MethodPost, cfg.HTTPHost, file)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("key", key)
	req.Header.Add("filename", fileName)
	req.Header.Add("command", FILE)
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(r))
}

func (c *client) read() {
	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadString(endl)
		if err != nil {
			log.Println(err)
			c.connect()
			return
		}
		line = line[:len(line)-1]
		go c.handleMessage(line)
	}
}

func (c *client) handleInput(line string) {
	spl := strings.SplitN(line, " ", 3)
	if len(spl) < 2 {
		log.Println("usage: nuuls Kappa 123")
		return
	}
	if inputCmd.MatchString(spl[0]) && len(spl) == 3 {
		cmd := spl[0][1:] // remove "/"
		switch cmd {
		case "m", "msg", "w":
			c.queueMessage(spl[1], spl[2])
		case "f", "file", "i":
			file, err := encryptFile(spl[1], spl[2])
			if err != nil {
				log.Println(err)
				return
			}
			defer file.Close()
			header := http.Header{}
			header.Add("targetuser", strings.ToLower(spl[1]))
			header.Add("command", FILE)
			log.Println(spl[2])
			header.Add("filename", getFileName(spl[2]))
			c.queue(file, header)
			log.Println("sent file", file.Name())
		}
		return
	}
	c.queueMessage(spl[0], strings.Join(spl[1:], " "))

}

func (c *client) readInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
		}
		// remove "\n"
		msg = msg[:len(msg)-2]
		go c.handleInput(msg)
	}
}