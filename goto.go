package main

import (
	"encoding/json"
	"errors"
	"fmt"
	plug "github.com/jcline/goto/plugins"
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

type UriFunc func(*string) (*string, error)
type WriteFunc func(*plug.IRCMessage, *string) (*plug.IRCMessage, error)

// Commands
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
	Server   string          `json:"server"`
	UserName string          `json:"userName"`
	RealName string          `json:"realName"`
	Channels []string        `json:"channels"`
	Plugins  plug.PluginConf `json:"plugin_conf"`
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

func getStr(prompt string, failurePrompt string, invalid func(string) bool) (result string, err error) {
	result = ""
	for {
		log.Println(prompt)
		_, err = fmt.Scanf("%s", &result)
		if err != nil {
			return
		}
		if invalid(result) {
			log.Println(failurePrompt)
		} else {
			return
		}
	}
}

func createConfig(path string, useMal bool, useYoutube bool) (conf Settings, err error) {

	_, err = exists(path)
	log.Println(exists, err)
	if err == ErrConfNotFound {
		err = os.MkdirAll(filepath.Dir(path), 0755)
		log.Println(path, ":", filepath.Dir(path))
		if err != nil && !os.IsPermission(err) {
			return
		}
	}

	isEmpty := func(s string) bool {
		return strings.TrimSpace(s) == ""
	}

	conf.Server, err = getStr("Server (e.g. irc.freenode.net:6666):", "You must include a port.", func(s string) bool { return !strings.Contains(s, ":") })
	if err != nil {
		return
	}

	conf.UserName, err = getStr("User name:", "User name must not be empty", isEmpty)
	if err != nil {
		return
	}

	conf.RealName, err = getStr("Real name:", "Real name must not be empty", isEmpty)
	if err != nil {
		return
	}

	chans, err := getStr("Channels to join (e.g. #chan1,#chan2 or #chan1):", "You must provide at least one channel", func(s string) bool {
		return isEmpty(s) || !strings.Contains(s, "#")
	})
	if err != nil {
		return
	}
	conf.Channels = strings.Split(chans, ",")

	if useMal {
		conf.Plugins.Mal.User, err = getStr("MAL user name:", "User name must not be empty", isEmpty)
		if err != nil {
			return
		}
		conf.Plugins.Mal.Password, err = getStr("MAL password:", "Password must not be empty", isEmpty)
		if err != nil {
			return
		}
		conf.Plugins.Mal.UserAgent, err = getStr("MAL User Agent:", "User Agent must not be empty", isEmpty)
		if err != nil {
			return
		}
	}

	if useYoutube {
		conf.Plugins.Youtube.Key, err = getStr("Youtube API Key:", "API key must not be empty", isEmpty)
		if err != nil {
			return
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
				conf, err = createConfig(path, true, true)
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

	log.Println(conf)
	conf.Plugins.UserName = conf.UserName
	con, err := goty.Dial(conf.Server, conf.UserName, conf.RealName)
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	writeMessage := make(chan plug.IRCMessage, 1000)
	go messageHandler(con, writeMessage, conf.Channels, 10, 2)

	var autobanner plug.Autoban
	var plugins []plug.Plugin
	plugins = append(plugins, new(plug.AmiAmi))
	plugins = append(plugins, new(plug.Help))
	plugins = append(plugins, new(plug.Mal))
	plugins = append(plugins, new(plug.MyFigureCollection))
	plugins = append(plugins, new(plug.Plamoya))
	plugins = append(plugins, new(plug.Reddit))
	plugins = append(plugins, new(plug.Vimeo))
	plugins = append(plugins, new(plug.Youtube))

	autobanner.Setup(writeMessage, conf.Plugins)
	for _, plugin := range plugins {
		plugin.Setup(writeMessage, conf.Plugins)
	}

	auth(con, writeMessage, conf.UserName)
	for _, channel := range conf.Channels {
		con.Write <- "JOIN " + channel
	}

	for msg := range con.Read {
		log.Println(msg)
		prepared, err := getMsgInfo(msg)
		if err != nil {
			continue
		}
		prepared.When = time.Now()

		if autobanner.Match(prepared) {
			continue
		}

		// half assed filtering
		_, notFound := plug.GetFirstMatch(matchSpoilers, &prepared.Msg)
		if notFound != nil {
			for _, plugin := range plugins {
				if plugin.Match(prepared) {
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
		if !msg.Unlimited && now.Sub(allBooks[key]) < time.Duration(delay)*time.Second { //|| now.Sub(chanBooks[key]) < time.Second*2 {
			continue
		}
		allBooks[key] = now
		//chanBooks[key] = now
		message(con, msg)
	}
}

var PRIVMSG = regexp.MustCompile(`:(.+![^ ]+) PRIVMSG ([^ ]+) :(.*)`)
var USER = regexp.MustCompile(`([^!]+)![^@]+@(.+)`)

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

	userMask := match[0][1]
	userMatch := USER.FindAllStringSubmatch(userMask, -1)

	imsg.User = userMatch[0][1]
	imsg.Mask = userMatch[0][2]
	imsg.Channel = match[0][2]
	imsg.Msg = match[0][3]
	return imsg, nil
}
