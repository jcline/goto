package main

import (
	plug "./plugins"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jcline/goty"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var user = ""

type UriFunc func(*string) (*string, error)
type WriteFunc func(*plug.IRCMessage, *string) (*plug.IRCMessage, error)

// Commands
var matchHelp = regexp.MustCompile(`^help`)
var matchHelpTerms = regexp.MustCompile(`^help (.+)`)
var matchSpoilers = regexp.MustCompile(`(?i)(.*spoil.*)`)

func auth(con *goty.IRCConn, writeMessage chan plug.IRCMessage, user string) {
	var pswd string
	fmt.Printf("Password for NickServ:\n")
	_, err := fmt.Scanf("%s", &pswd)
	if err != nil {
		return
	}

	msg := plug.IRCMessage{Channel: "NickServ", Msg: "IDENTIFY " + user + " " + pswd}
	writeMessage <- msg
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, ErrConfNotFound
	}
	return false, err
}

type Settings struct {
	Server   string   `json:"server"`
	UserName string   `json:"userName"`
	RealName string   `json:"realName"`
	Channels []string `json:"channels"`
}

var ErrConfNotFound = errors.New("Conf does not exist")

func readConfig() (conf Settings, path string, err error) {
	args := os.Args
	path = ""
	if len(args) == 2 {
		path = filepath.Clean(args[1])
	} else {
		path = os.Getenv("XDG_CONFIG_HOME")
		if path == "" {
			path = filepath.Join("$HOME", ".config", "goto", "conf")
		} else {
			path = filepath.Join(path, "goto", "conf")
		}
	}

	path, err = filepath.Abs(os.ExpandEnv(path))
	if err != nil {
		return
	}

	log.Println(path)

	_, err = exists(path)
	if err != nil {
		return
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = json.Unmarshal(file, &conf)
	if err != nil {
		return
	}

	return
}

func createConfig(path string) (conf Settings, err error) {

	_, err = exists(path)
	log.Println(exists, err)
	if err == ErrConfNotFound {
		err = os.MkdirAll(filepath.Dir(path), 0644)
		log.Println(path, ":", filepath.Dir(path))
		if err != nil && !os.IsPermission(err) {
			return
		}
	}

	for {
		log.Println("Server (e.g. irc.freenode.net:6666):")
		_, err = fmt.Scanf("%s", &conf.Server)
		if err != nil {
			return
		}
		if !strings.Contains(conf.Server, ":") {
			log.Println("You must include a port.")
		} else {
			break
		}
	}

	for {
		log.Println("User name:")
		_, err = fmt.Scanf("%s", &conf.UserName)
		if err != nil {
			return
		}
		if conf.UserName == "" {
			log.Println("User name must not be empty")
		} else {
			break
		}
	}

	for {
		log.Println("Real name:")
		_, err = fmt.Scanf("%s", &conf.RealName)
		if err != nil {
			return
		}
		if conf.RealName == "" {
			log.Println("Real name must not be empty")
		} else {
			break
		}
	}

	for {
		log.Println("Channels to join (e.g. #chan1,#chan2 or #chan1):")
		var channels string
		_, err = fmt.Scanf("%s", &channels)
		if err != nil {
			return
		}
		if channels == "" || !strings.Contains(channels, "#") {
			log.Println("You must provide at least one channel")
		} else {
			conf.Channels = strings.Split(channels, ",")
			break
		}
	}

	js, err := json.Marshal(conf)
	if err != nil {
		return
	}

	log.Println("Writing to: ", path)
	err = ioutil.WriteFile(path, js, 0644)
	if err != nil {
		return
	}

	return
}

func main() {
	conf, path, err := readConfig()
	if err != nil {
		if err == ErrConfNotFound {
			log.Println("Could not read config, would you like to create one? [y/n]")
			var response string
			_, err := fmt.Scanf("%s", &response)
			if err != nil {
				log.Fatal(err)
			}
			if response == "y" || response == "Y" {
				conf, err = createConfig(path)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal("I can't do anything without config.")
			}
		} else {
			log.Fatal(err)
		}
	}

	user = conf.UserName
	con, err := goty.Dial(conf.Server, conf.UserName, conf.RealName)
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	writeMessage := make(chan plug.IRCMessage, 1000)
	go messageHandler(con, writeMessage, conf.Channels, 10, 2)

	var plugins []plug.Plugin
	plugins = append(plugins, new(plug.Youtube))
	plugins = append(plugins, new(plug.AmiAmi))
	plugins = append(plugins, new(plug.Reddit))
	plugins = append(plugins, new(plug.Mal))

	for _, plugin := range plugins {
		plugin.Setup(writeMessage)
	}

	auth(con, writeMessage, conf.UserName)
	for _, channel := range conf.Channels {
		con.Write <- "JOIN " + channel
	}

	for msg := range con.Read {
		log.Printf("%s\n", msg)
		prepared, err := getMsgInfo(msg)
		if err != nil {
			continue
		}
		prepared.When = time.Now()

		// half assed filtering
		_, notFound := getFirstMatch(matchSpoilers, &prepared.Msg)
		if notFound != nil {
			for _, plugin := range plugins {
				if plugin.Match().MatchString(prepared.Msg) {
					plugin.Event() <- *prepared
				}
			}
		}
	}
	con.Close()
}

type unparsedMessage struct {
	msg  string
	when time.Time
}

func message(con *goty.IRCConn, msg plug.IRCMessage) {
	privmsg := "PRIVMSG " + msg.Channel + " :" + msg.Msg + "\r\n"
	log.Println(privmsg)
	con.Write <- privmsg
}

func messageHandler(con *goty.IRCConn, event chan plug.IRCMessage, channels []string, chanDelay, pmDelay int) {
	allBooks := map[string]time.Time{}
	//chanBooks := map[string]time.Time{}
	for msg := range event {
		now := time.Now()
		key := msg.Channel + ":" + msg.User
		delay := pmDelay
		for _, channel := range channels {
			if msg.Channel == channel {
				delay = chanDelay
				break
			}
		}
		if now.Sub(allBooks[key]) < time.Duration(delay)*time.Second { //|| now.Sub(chanBooks[key]) < time.Second*2 {
			continue
		}
		allBooks[key] = now
		//chanBooks[key] = now
		message(con, msg)
	}
}

var PRIVMSG = regexp.MustCompile(`:(.+)![^ ]+ PRIVMSG ([^ ]+) :(.*)`)

func getMsgInfo(msg string) (*plug.IRCMessage, error) {
	// :nick!~realname@0.0.0.0 PRIVMSG #chan :msg
	imsg := new(plug.IRCMessage)
	match := PRIVMSG.FindAllStringSubmatch(msg, -1)
	if len(match) < 1 {
		return imsg, errors.New("could not parse message")
	}
	if len(match[0]) < 3 {
		return imsg, errors.New("could not parse message")
	}
	imsg.User = user
	imsg.Channel = match[0][2]
	if imsg.Channel == user {
		imsg.Channel = match[0][1]
	}
	imsg.Msg = match[0][3]
	return imsg, nil
}

func getFirstMatch(re *regexp.Regexp, matchee *string) (*string, error) {
	match := re.FindAllStringSubmatch(*matchee, -1)
	if len(match) < 1 {
		return nil, errors.New("Could not match")
	}
	if len(match[0]) < 2 {
		return nil, errors.New("Could not match")
	}
	return &match[0][1], nil
}
