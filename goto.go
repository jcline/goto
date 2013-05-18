package main

import (
	"encoding/xml" // gelbooru parsing
	"errors"
	"fmt"
	"github.com/jcline/goty"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"
)

var user = ""

var matchGelbooru = regexp.MustCompile(`.*\Qhttp://gelbooru.com/index.php?page=post&s=view&id=\E([\d]+).*`)
var matchYouTube = regexp.MustCompile(`.*(https?://(?:www\.|)youtu(?:\.be|be\.com)/[^ ]+).*`)
var matchAmiAmi = regexp.MustCompile(`(https?://(?:www\.|)amiami.com/[^/]+/detail/.*)`)

func auth(con *goty.IRCConn, writeMessage chan IRCMessage) {
	var pswd string
	_, err := fmt.Scanf("%s", &pswd)
	if err != nil {
		os.Exit(1)
	}

	msg := IRCMessage{"NickServ", "IDENTIFY Laala " + pswd}
	writeMessage <- msg
}

func main() {
	args := os.Args
	if len(args) < 4 {
		os.Exit(1)
	}

	con, err := goty.Dial(args[1], args[2], args[3])
	user = args[2]
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
	}

	gelbooruEvent := make(chan string, 1000)
	youtubeEvent := make(chan string, 1000)
	amiAmiEvent := make(chan string, 1000)
	BastilleEvent := make(chan string, 1000)
	writeMessage := make(chan IRCMessage, 1000)

	go messageHandler(con, writeMessage)
	go gelbooru(con, gelbooruEvent, writeMessage)
	go youtube(con, youtubeEvent, writeMessage)
	go amiami(con, amiAmiEvent, writeMessage)
	go Bastille(con, BastilleEvent, writeMessage)

	auth(con, writeMessage)
	con.Write <- "JOIN " + args[4]

	for {
		msg, ok := <-con.Read
		if !ok {
			break
		}
		fmt.Printf("%s\n", msg)

		switch {
		case matchGelbooru.MatchString(msg):
			//gelbooruEvent <- matchGelbooru.FindAllStringSubmatch(msg, -1)[0][1]
		case matchYouTube.MatchString(msg):
			youtubeEvent <- msg
		case matchAmiAmi.MatchString(msg):
			amiAmiEvent <- msg
		default:
		}
	}
	con.Close()
}

type IRCMessage struct {
	channel string
	msg     string
}

func message(con *goty.IRCConn, msg IRCMessage) {
	con.Write <- "PRIVMSG " + msg.channel + " :" + msg.msg + "\r\n"
}

func messageHandler(con *goty.IRCConn, event chan IRCMessage) {
	books := map[string]time.Time{}
	for msg := range event {
		now := time.Now()
		if now.Sub(books[msg.channel]) < time.Second*10 {
			continue
		}
		books[msg.channel] = now
		message(con, msg)
	}
}

var PRIVMSG = regexp.MustCompile(`:(.+)![^ ]+ PRIVMSG ([^ ]+) :(.*)`)

func getChannel(msg string) (*IRCMessage, error) {
	// :nick!~realname@0.0.0.0 PRIVMSG #chan :msg
	imsg := new(IRCMessage)
	match := PRIVMSG.FindAllStringSubmatch(msg, -1)
	if len(match) < 1 {
		return imsg, errors.New("could not parse message")
	}
	if len(match[0]) < 3 {
		return imsg, errors.New("could not parse message")
	}
	imsg.channel = match[0][2]
	if imsg.channel == user {
		imsg.channel = match[0][1]
	}
	imsg.msg = match[0][3]
	return imsg, nil
}

func Bastille(con *goty.IRCConn, event chan string, writeMessage chan IRCMessage) {
	msgs := []string{
		"Bastille, yo brodudedudebro!!!!1",
		"Bastille, wat up homie",
		"Bastille, word",
		"Bastille, duuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuude",
		"'sup Bastille?",
	}

	for msg := range event {
		parsed, err := getChannel(msg)
		if err != nil {
			continue
		}
		writeMessage <- IRCMessage{parsed.channel, msgs[rand.Intn(len(msgs))-1]}
	}
}

func amiami(con *goty.IRCConn, event chan string, writeMessage chan IRCMessage) {
	matchTitle := regexp.MustCompile(`.*<meta property="og:title" content="(.+)" />.*`)
	matchDiscount := regexp.MustCompile(`[0-9]+\%OFF `)
	for msg := range event {
		parsed, err := getChannel(msg)
		if err != nil {
			continue
		}

		resp, err := http.Get(matchAmiAmi.FindAllStringSubmatch(parsed.msg, -1)[0][1])
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

		writeMessage <- IRCMessage{parsed.channel, matchDiscount.ReplaceAllLiteralString(matchTitle.FindAllStringSubmatch(string(body), -1)[0][1], "")}
	}
}

func youtube(con *goty.IRCConn, event chan string, writeMessage chan IRCMessage) {
	matchTitle := regexp.MustCompile(`.*<title>(.+)</title>.*`)
	matchUser := regexp.MustCompile(`.*<a class=".*yt-user-name[^>]+>(.+)</a>.*`)
	for msg := range event {
		parsed, err := getChannel(msg)
		if err != nil {
			continue
		}

		resp, err := http.Get(matchYouTube.FindAllStringSubmatch(parsed.msg, -1)[0][1])
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

		writeMessage <- IRCMessage{parsed.channel, matchTitle.FindAllStringSubmatch(string(body), -1)[0][1] + " by " +
			matchUser.FindAllStringSubmatch(string(body), -1)[0][1]}
	}
}

func gelbooru(con *goty.IRCConn, event chan string, writeMessage chan IRCMessage) {
	type Post struct {
		post string
		tags string `xml:",attr"`
	}

	for msg := range event {
		parsed, err := getChannel(msg)
		if err != nil {
			continue
		}

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
		writeMessage <- IRCMessage{parsed.channel, "tobedone"}
	}
}
