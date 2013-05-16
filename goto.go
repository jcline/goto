package main

import (
	"github.com/RecursiveForest/goty" //fork and maintain building copy
	"encoding/xml" // gelbooru parsing
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
)

var matchGelbooru = regexp.MustCompile(`.*\Qhttp://gelbooru.com/index.php?page=post&s=view&id=\E([\d]+).*`)
var matchYouTube = regexp.MustCompile(`.*(https?://(?:www\.|)youtu(?:\.be|be\.com)/[^ ]+).*`)

func main() {
	args := os.Args
	if len(args) < 4 { os.Exit(1) }

	con, err := goty.Dial(args[1], args[2], args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
	}
	//con.Write <- "JOIN #reddit-anime"
	con.Write <- "JOIN " + args[4]

	gelbooruEvent := make(chan string, 1000)
	youtubeEvent := make(chan string, 1000)
	writeMessage := make(chan IRCMessage, 1000)
	go messageHandler(con, writeMessage)
	go gelbooru(con, gelbooruEvent, writeMessage)
	go youtube(con, youtubeEvent, writeMessage)

	for {
		msg, ok := <-con.Read
		if !ok { break }
		fmt.Printf("%s\n", msg)

		switch {
		case matchGelbooru.MatchString(msg):
			gelbooruEvent <- matchGelbooru.FindAllStringSubmatch(msg, -1)[0][1]
		case matchYouTube.MatchString(msg):
			youtubeEvent <- matchYouTube.FindAllStringSubmatch(msg, -1)[0][1]
		default:

		}
	}
	con.Close()
}

type IRCMessage struct {
	channel string
	msg string
}

func message(con *goty.IRCConn, msg IRCMessage) {
	con.Write <- "PRIVMSG " + msg.channel + " :" + msg.msg + "\r\n"
}

func messageHandler(con *goty.IRCConn, event chan IRCMessage) {
	throttle := time.Tick(time.Second*5)
	for msg := range event {
		message(con, msg)
		<-throttle
	}
}

func youtube(con *goty.IRCConn, event chan string, writeMessage chan IRCMessage) {
	matchTitle := regexp.MustCompile(`.*<title>(.+)(?:- YouTube)?</title>.*`)
	for msg := range event {
		resp, err := http.Get(msg)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		fmt.Printf("%s\n", matchTitle.FindAllStringSubmatch(string(body), -1)[0][1])
		writeMessage <- IRCMessage{"##jtest517", matchTitle.FindAllStringSubmatch(string(body), -1)[0][1]}
	}
}

func gelbooru(con *goty.IRCConn, event chan string, writeMessage chan IRCMessage) {
	type Post struct {
		post string
		tags string `xml:",attr"`
	}

	for {
		msg, ok := <-event
		if !ok { break }

		resp, err := http.Get("http://gelbooru.com/index.php?page=dapi&s=post&q=index&tags&id=" + msg)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		fmt.Printf("%s\n", body)

		var result Post

		err = xml.Unmarshal(body, &result)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		fmt.Printf("%s\n", result.tags)
	}
}
