package main

import (
	"encoding/json"
	"encoding/xml" // gelbooru parsing
	"errors"
	"fmt"
	"github.com/jcline/DamerauLevenshteinDistance"
	"github.com/jcline/goty"
	"html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var user = ""

var matchHelp = regexp.MustCompile(`^help`)
var matchHelpTerms = regexp.MustCompile(`^help (.+)`)
var matchAniDBSearch = regexp.MustCompile(`!anidb +(.+) *`)
var matchAmiAmi = regexp.MustCompile(`(?:https?://|)(?:www\.|)amiami.com/((?:[^/]|\S)+/detail/\S+)`)
var matchGelbooru = regexp.MustCompile(`(?:https?://|)\Qgelbooru.com/index.php?page=post&s=view&id=\E([\d]+)`)
var matchMALAnime = regexp.MustCompile(`^!anime (.+)`)
var matchMALManga = regexp.MustCompile(`^!manga (.+)`)
var matchReddit = regexp.MustCompile(`(?:http://|)(?:www\.|https://pay\.|)redd(?:\.it|it\.com)/(?:r/(?:[^/ ]|\S)+/comments/|)([a-z0-9]{6})/?(?:[ .]+|\z)`)
var matchYouTube = regexp.MustCompile(`(?:https?://|)(?:www\.|)(youtu(?:\.be|be\.com)/\S+)`)

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

	writeMessage := make(chan IRCMessage, 1000)
	go messageHandler(con, writeMessage, conf.Channels, 10, 2)

	amiAmiEvent := make(chan IRCMessage, 1000)
	//anidbEvent := make(chan IRCMessage, 1000)
	bastilleEvent := make(chan IRCMessage, 1000)
	gelbooruEvent := make(chan IRCMessage, 1000)
	animeEvent := make(chan IRCMessage, 1000)
	mangaEvent := make(chan IRCMessage, 1000)
	redditEvent := make(chan IRCMessage, 1000)
	youtubeEvent := make(chan IRCMessage, 1000)
	helpEvent := make(chan IRCMessage, 1000)

	go func() {
		for {
			duration := time.Duration(rand.Int63n(2*96)) * time.Hour
			log.Println("Waiting ", duration)
			time.Sleep(duration)
			bastilleEvent <- IRCMessage{"#reddit-anime", "", "", time.Now()}
		}
	}()
	go bastille(bastilleEvent, writeMessage)

	go amiami(amiAmiEvent, writeMessage)
	//go anidb(anidbEvent, writeMessage)
	go gelbooru(gelbooruEvent, writeMessage)
	go malSearch(animeEvent, "anime", writeMessage, matchMALAnime)
	go malSearch(mangaEvent, "manga", writeMessage, matchMALManga)
	go reddit(redditEvent, writeMessage)
	go youtube(youtubeEvent, writeMessage)
	go help(helpEvent, writeMessage, conf.Channels)

	auth(con, writeMessage, conf.UserName)
	for _, channel := range conf.Channels {
		con.Write <- "JOIN " + channel
	}

	for msg := range con.Read {
		log.Printf("%s\n", msg)
		prepared, err := getMsgInfo(msg)
		if err != nil {
			//log.Printf("%v\n", err)
			continue
		}
		prepared.when = time.Now()

		switch {
		case matchAmiAmi.MatchString(prepared.msg):
			amiAmiEvent <- *prepared
		case matchGelbooru.MatchString(prepared.msg):
			//gelbooruEvent <- matchGelbooru.FindAllStringSubmatch(prepared.msg, -1)[0][1]
		case matchMALAnime.MatchString(prepared.msg):
			animeEvent <- *prepared
		case matchMALManga.MatchString(prepared.msg):
			mangaEvent <- *prepared
		case matchReddit.MatchString(prepared.msg):
			redditEvent <- *prepared
		case matchYouTube.MatchString(prepared.msg):
			youtubeEvent <- *prepared
		case matchHelp.MatchString(prepared.msg):
			helpEvent <- *prepared
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
	when    time.Time
}

func message(con *goty.IRCConn, msg IRCMessage) {
	privmsg := "PRIVMSG " + msg.channel + " :" + msg.msg + "\r\n"
	log.Println(privmsg)
	con.Write <- privmsg
}

func messageHandler(con *goty.IRCConn, event chan IRCMessage, channels []string, chanDelay, pmDelay int) {
	allBooks := map[string]time.Time{}
	//chanBooks := map[string]time.Time{}
	for msg := range event {
		now := time.Now()
		key := msg.channel + ":" + msg.user
		delay := pmDelay
		for _, channel := range channels {
			if msg.channel == channel {
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

func bastille(event chan IRCMessage, writeMessage chan IRCMessage) {
	msgs := []string{
		"Bastille, yo brodudedudebro!!!!1",
		"Bastille, wat up homie",
		"Bastille, word",
		"Bastille, duuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuude",
		"'sup Bastille?",
	}

	for msg := range event {
		writeMessage <- IRCMessage{msg.channel, msgs[rand.Intn(len(msgs))-1], msg.user, msg.when}
	}
}

type uriFunc func(*string) (*string, error)
type writeFunc func(*IRCMessage, *string) error
type errFunc func(*IRCMessage, error) error

func scrapeAndSend(event chan IRCMessage, findUri uriFunc, write writeFunc) {
	var f = func(msg IRCMessage) {
		uri, err := findUri(&msg.msg)
		if err != nil {
			log.Println(err)
			return
		}

		resp, err := http.Get(*uri)
		if err != nil {
			log.Println(err)
			return
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Println(err)
			return
		}
		body := string(bodyBytes)

		err = write(&msg, &body)
		if err != nil {
			log.Println(err)
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

func amiami(event chan IRCMessage, writeMessage chan IRCMessage) {
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

			writeMessage <- IRCMessage{msg.channel, "[AmiAmi] " + matchDiscount.ReplaceAllLiteralString(*title, ""), msg.user, msg.when}
			return nil
		})
}

func reddit(event chan IRCMessage, writeMessage chan IRCMessage) {
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

			cleanTitle := html.UnescapeString(*title)
			if cleanTitle != "reddit.com: page not found" {
				writeMessage <- IRCMessage{msg.channel, "[Reddit] " + cleanTitle, msg.user, msg.when}
			} else {
				return errors.New("Page not found")
			}
			return nil
		})
}

func youtube(event chan IRCMessage, writeMessage chan IRCMessage) {
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
			writeMessage <- IRCMessage{msg.channel, "[YouTube] " + html.UnescapeString(*title+" uploaded by "+*user), msg.user, msg.when}
			return nil
		})
}

func gelbooru(event chan IRCMessage, writeMessage chan IRCMessage) {
	type Post struct {
		post string
		tags string `xml:",attr"`
	}

	for msg := range event {
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
		writeMessage <- IRCMessage{msg.channel, "tobedone", msg.user, msg.when}
	}
}

func anidb(event chan IRCMessage, writeMessage chan IRCMessage) {
	cache := make(map[string]string)

	for msg := range event {
		val, ok := cache[msg.msg]
		if ok {
			writeMessage <- IRCMessage{msg.channel, val, msg.user, msg.when}
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

type Results []Result
type Result struct {
	Id             int
	Title          string
	Classification string
	search         string
	computed       bool
	distance       int
}

func (r Results) Len() int {
	return len(r)
}

func (r Results) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Results) Less(i, j int) bool {
	if !r[i].computed {
		r[i].distance = DamerauLevenshteinDistance.Distance(r[i].search, r[i].Title)
		r[i].computed = true
	}
	if !r[j].computed {
		r[j].distance = DamerauLevenshteinDistance.Distance(r[j].search, r[j].Title)
		r[j].computed = true
	}

	return r[i].distance < r[j].distance
}

func malSearch(event chan IRCMessage, searchType string, writeMessage chan IRCMessage, match *regexp.Regexp) {
	var terms *string
	var err error
	scrapeAndSend(event, func(msg *string) (*string, error) {
		terms, err = getFirstMatch(match, msg)
		if err != nil {
			return nil, err
		}
		uri := "http://mal-api.com/" + searchType + "/search?q=" + url.QueryEscape(*terms)
		return &uri, nil
	},
		func(msg *IRCMessage, body *string) error {
			if len(*body) < 10 {
				writeMessage <- IRCMessage{msg.channel, "┐('～`；)┌", msg.user, msg.when}
				return errors.New("No results")
			}

			var r Results
			err := json.Unmarshal([]byte(*body), &r)
			if err != nil {
				writeMessage <- IRCMessage{msg.channel, "┐('～`；)┌", msg.user, msg.when}
				return err
			}
			fmt.Printf("%v\n", r)

			var results = ""
			var nsfw = false
			reference, _ := getFirstMatch(match, &msg.msg)

			for i, _ := range r {
				r[i].Title = html.UnescapeString(r[i].Title)
				r[i].search = *reference
				r[i].computed = false
			}
			sort.Sort(r)

			length := 2
			if len(r) < length {
				length = len(r)
			}
			for count, result := range r {
				if searchType == "anime" {
					class := result.Classification
					if class != "" {
						if strings.Contains(class, "Rx") ||
							strings.Contains(class, "R+") ||
							strings.Contains(class, "Hentai") {
							nsfw = true
						}
						class = " [Rating " + class + "]"
					} else {
						nsfw = true
					}

					results += result.Title + class + " http://myanimelist.net/" + searchType + "/" + strconv.Itoa(result.Id) + "  "
				} else {
					results += result.Title + " http://myanimelist.net/" + searchType + "/" + strconv.Itoa(result.Id) + "  "
					nsfw = true
				}
				if count >= length {
					break
				}
			}

			if nsfw {
				results = "NSFW " + results
			}

			if len(r) > 3 {
				results += "More: " + "http://myanimelist.net/" + searchType + ".php?q=" + url.QueryEscape(*terms)
			}

			writeMessage <- IRCMessage{msg.channel, results, msg.user, msg.when}
			return nil
		})
}

func help(event chan IRCMessage, writeMessage chan IRCMessage, excludedChannels []string) {
	for msg := range event {
		stop := false
		for _, channel := range excludedChannels {
			if msg.channel == channel {
				stop = true
				break
			}
		}
		if stop {
			continue
		}
		helpMsg := IRCMessage{channel: msg.channel, user: msg.user, when: msg.when}
		terms, err := getFirstMatch(matchHelpTerms, &msg.msg)
		if err != nil {
			helpMsg.msg = "Usage: help mal"
			writeMessage <- helpMsg
			continue
		}
		switch {
		case *terms == "mal":
			helpMsg.msg = "MAL search: !anime lain | !manga monster"
		}
		writeMessage <- helpMsg
	}
}
