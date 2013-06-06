package main

import (
	"encoding/json"
	"encoding/xml" // gelbooru parsing
	"errors"
	"fmt"
	"github.com/jcline/goty"
	"html"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

var user = ""

var matchAniDBSearch = regexp.MustCompile(`!anidb +(.+) *`)
var matchAmiAmi = regexp.MustCompile(`(?:https?://|)(?:www\.|)amiami.com/([^/ ]+/detail/[^ ]+)`)
var matchGelbooru = regexp.MustCompile(`(?:https?://|)\Qgelbooru.com/index.php?page=post&s=view&id=\E([\d]+)`)
var matchMAL = regexp.MustCompile(`!anime+(.+)`)
var matchReddit = regexp.MustCompile(`(?:http://|)(?:www\.|https://pay\.|)redd(?:\.it|it\.com)/(r/[^/ ]+/comments/[^/ ]+)/?(?: .*|\z)`)
var matchYouTube = regexp.MustCompile(`(?:https?://|)(?:www\.|)(youtu(?:\.be|be\.com)/[^ ]+)`)

func auth(con *goty.IRCConn, writeMessage chan IRCMessage, user string) {
	var pswd string
	fmt.Printf("Password for NickServ:\n")
	_, err := fmt.Scanf("%s", &pswd)
	if err != nil {
		return
	}

	msg := IRCMessage{channel: "NickServ", msg: "IDENTIFY " + user + " " + pswd}
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

	writeMessage := make(chan IRCMessage, 1000)
	go messageHandler(con, writeMessage)

	amiAmiEvent := make(chan unparsedMessage, 1000)
	//anidbEvent := make(chan unparsedMessage, 1000)
	bastilleEvent := make(chan unparsedMessage, 1000)
	gelbooruEvent := make(chan unparsedMessage, 1000)
	malEvent := make(chan unparsedMessage, 1000)
	redditEvent := make(chan unparsedMessage, 1000)
	youtubeEvent := make(chan unparsedMessage, 1000)

	go amiami(amiAmiEvent, writeMessage)
	//go anidb(anidbEvent, writeMessage)
	go bastille(bastilleEvent, writeMessage)
	go gelbooru(gelbooruEvent, writeMessage)
	go malSearch(malEvent, writeMessage)
	go reddit(redditEvent, writeMessage)
	go youtube(youtubeEvent, writeMessage)

	auth(con, writeMessage, user)
	con.Write <- "JOIN " + args[4]

	for msg := range con.Read {
		prepared := unparsedMessage{msg, time.Now()}
		fmt.Printf("%s||%s\n", prepared.when, prepared.msg)

		switch {
		case matchAmiAmi.MatchString(prepared.msg):
			amiAmiEvent <- prepared
		case matchGelbooru.MatchString(prepared.msg):
			//gelbooruEvent <- matchGelbooru.FindAllStringSubmatch(prepared.msg, -1)[0][1]
		case matchMAL.MatchString(prepared.msg):
			malEvent <- prepared
		case matchReddit.MatchString(prepared.msg):
			redditEvent <- prepared
		case matchYouTube.MatchString(prepared.msg):
			youtubeEvent <- prepared
		//case matchAniDBSearch.MatchString(prepared.msg):
		//anidbEvent <- prepared
		default:
		}
	}
	con.Close()
}

type unparsedMessage struct {
	msg  string
	when time.Time
}

type IRCMessage struct {
	channel string
	msg     string
	user    string
}

func message(con *goty.IRCConn, msg IRCMessage) {
	con.Write <- "PRIVMSG " + msg.channel + " :" + msg.msg + "\r\n"
}

func messageHandler(con *goty.IRCConn, event chan IRCMessage) {
	allBooks := map[string]time.Time{}
	chanBooks := map[string]time.Time{}
	for msg := range event {
		now := time.Now()
		key := msg.channel + ":" + msg.user
		if now.Sub(allBooks[key]) < time.Second*10 || now.Sub(chanBooks[key]) < time.Second*2 {
			continue
		}
		allBooks[key] = now
		chanBooks[key] = now
		message(con, msg)
	}
}

var PRIVMSG = regexp.MustCompile(`:(.+)![^ ]+ PRIVMSG ([^ ]+) :(.*)`)

func getMsgInfo(msg string) (*IRCMessage, error) {
	// :nick!~realname@0.0.0.0 PRIVMSG #chan :msg
	imsg := new(IRCMessage)
	match := PRIVMSG.FindAllStringSubmatch(msg, -1)
	if len(match) < 1 {
		return imsg, errors.New("could not parse message")
	}
	if len(match[0]) < 3 {
		return imsg, errors.New("could not parse message")
	}
	imsg.user = user
	imsg.channel = match[0][2]
	if imsg.channel == user {
		imsg.channel = match[0][1]
	}
	imsg.msg = match[0][3]
	return imsg, nil
}

func bastille(event chan unparsedMessage, writeMessage chan IRCMessage) {
	msgs := []string{
		"Bastille, yo brodudedudebro!!!!1",
		"Bastille, wat up homie",
		"Bastille, word",
		"Bastille, duuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuude",
		"'sup Bastille?",
	}

	for msg := range event {
		parsed, err := getMsgInfo(msg.msg)
		if err != nil {
			continue
		}
		writeMessage <- IRCMessage{parsed.channel, msgs[rand.Intn(len(msgs))-1], parsed.user}
	}
}

type uriFunc func(*string) (*string, error)
type writeFunc func(*IRCMessage, *string) error
type errFunc func(*IRCMessage, error) error

func scrapeAndSend(event chan unparsedMessage, findUri uriFunc, write writeFunc) {
	var f = func(msg unparsedMessage) {
		parsed, err := getMsgInfo(msg.msg)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		uri, err := findUri(&parsed.msg)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		resp, err := http.Get(*uri)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		body := string(bodyBytes)

		err = write(parsed, &body)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
	}

	for msg := range event {
		go f(msg)
	}
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

func amiami(event chan unparsedMessage, writeMessage chan IRCMessage) {
	matchTitle := regexp.MustCompile(`.*<meta property="og:title" content="(.+)" />.*`)
	matchDiscount := regexp.MustCompile(`[0-9]+\%OFF `)
	scrapeAndSend(event, func(msg *string) (*string, error) {
		uri, err := getFirstMatch(matchAmiAmi, msg)
		if err != nil {
			return nil, err
		}

		fullUri := "http://amiami.com/" + *uri
		return &fullUri, nil
	},
		func(msg *IRCMessage, body *string) error {
			title, err := getFirstMatch(matchTitle, body)
			if err != nil {
				return err
			}

			writeMessage <- IRCMessage{msg.channel, "[AmiAmi]: " + matchDiscount.ReplaceAllLiteralString(*title, ""), msg.user}
			return nil
		})
}

func reddit(event chan unparsedMessage, writeMessage chan IRCMessage) {
	matchTitle := regexp.MustCompile(`.*<title>(.+)</title>.*`)

	scrapeAndSend(event, func(msg *string) (*string, error) {
		uri, err := getFirstMatch(matchReddit, msg)
		if err != nil {
			return nil, err
		}

		fullUri := "http://reddit.com/" + *uri
		return &fullUri, nil
	},
		func(msg *IRCMessage, body *string) error {
			title, err := getFirstMatch(matchTitle, body)
			if err != nil {
				return err
			}

			writeMessage <- IRCMessage{msg.channel, "[Reddit]: " + html.UnescapeString(*title), msg.user}
			return nil
		})
}

func youtube(event chan unparsedMessage, writeMessage chan IRCMessage) {
	matchTitle := regexp.MustCompile(`.*<title>(.+)(?: - YouTube){1}</title>.*`)
	matchUser := regexp.MustCompile(`.*<a[^>]+class="[^"]+yt-user-name[^>]+>([^<]+)</a>.*`)

	scrapeAndSend(event, func(msg *string) (*string, error) {
		uri, err := getFirstMatch(matchYouTube, msg)
		if err != nil {
			return nil, err
		}

		fullUri := "http://" + *uri
		return &fullUri, nil
	},
		func(msg *IRCMessage, body *string) error {
			title, err := getFirstMatch(matchTitle, body)
			if err != nil {
				return err
			}
			user, err := getFirstMatch(matchUser, body)
			if err != nil {
				return err
			}
			writeMessage <- IRCMessage{msg.channel, "[YouTube]: " + html.UnescapeString(*title+" uploaded by "+*user), msg.user}
			return nil
		})
}

func gelbooru(event chan unparsedMessage, writeMessage chan IRCMessage) {
	type Post struct {
		post string
		tags string `xml:",attr"`
	}

	for msg := range event {
		parsed, err := getMsgInfo(msg.msg)
		if err != nil {
			continue
		}

		resp, err := http.Get("http://gelbooru.com/index.php?page=dapi&s=post&q=index&tags&id=" + msg.msg)
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
		writeMessage <- IRCMessage{parsed.channel, "tobedone", parsed.user}
	}
}

func anidb(event chan unparsedMessage, writeMessage chan IRCMessage) {
	cache := make(map[string]string)

	for msg := range event {
		parsed, err := getMsgInfo(msg.msg)
		if err != nil {
			continue
		}

		val, ok := cache[msg.msg]
		if ok {
			writeMessage <- IRCMessage{parsed.channel, val, parsed.user}
		} else {
			// totally broken :(
			resp, err := http.Get("http://anisearch.outrance.pl/index.php?task=search&query=" + msg.msg)
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

			//cache[msg.msg] = ""
		}
	}
}

func malSearch(event chan unparsedMessage, writeMessage chan IRCMessage) {

	type Anime struct {
		Id int
		Title string
		Classification string
	}

	scrapeAndSend(event, func(msg *string) (*string, error) {
		terms, err := getFirstMatch(matchMAL, msg)
		if err != nil {
			return nil, err
		}
		uri := "http://mal-api.com/anime/search?q=" + url.QueryEscape(*terms)
		return &uri, nil
	},
		func(msg *IRCMessage, body *string) error {
			if len(*body) < 10 {
				writeMessage <- IRCMessage{msg.channel, "┐('～`；)┌", msg.user}
				return errors.New("No results")
			}

			fmt.Printf("%v\n", *body)

			var a []Anime
			err := json.Unmarshal([]byte(*body), &a)
			if err != nil {
				writeMessage <- IRCMessage{msg.channel, "┐('～`；)┌", msg.user}
				return err
			}
			fmt.Printf("%v\n", a)

			var results = ""
			var count = 0
			var nsfw = false

			for count < len(a) && count < 3 {
				results += a[count].Title + " [Rating: " + a[count].Classification + "]: http://myanimelist.net/anime/" + strconv.Itoa(a[count].Id) + "  "
				if a[count].Classification == "Rx" {
					nsfw = true
				}
				count = count + 1
			}

			if nsfw {
				results += "NSFW"
			}

			writeMessage <- IRCMessage{msg.channel, results, msg.user}
			return nil
		})
}
