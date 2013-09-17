package main

import (
	"encoding/json"
	//"encoding/xml" // gelbooru parsing
	"errors"
	"fmt"
	//"github.com/jcline/DamerauLevenshteinDistance"
	"github.com/jcline/goty"
	//"html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	//"net/url"
	"os"
	"path/filepath"
	plug "./plugins"
	"regexp"
	//"sort"
	//"strconv"
	"strings"
	"time"
)

var user = ""

type UriFunc func(*string) (*string, error)
type WriteFunc func(*plug.IRCMessage, *string) (*plug.IRCMessage, error)

// Commands
var matchHelp = regexp.MustCompile(`^help`)
var matchHelpTerms = regexp.MustCompile(`^help (.+)`)

// Filters
var matchSpoilers = regexp.MustCompile(`(?i)(.*spoil.*)`)

var matchAniDBSearch = regexp.MustCompile(`!anidb +(.+) *`)
var matchAmiAmi = regexp.MustCompile(`(?:https?://|)(?:www\.|)amiami.com/((?:[^/]|\S)+/detail/\S+)`)
var matchGelbooru = regexp.MustCompile(`(?:https?://|)\Qgelbooru.com/index.php?page=post&s=view&id=\E([\d]+)`)
var matchMALAnime = regexp.MustCompile(`^!anime (.+)`)
var matchMALManga = regexp.MustCompile(`^!manga (.+)`)
var matchReddit = regexp.MustCompile(`(?:http://|)(?:www\.|https://pay\.|)redd(?:\.it|it\.com)/(?:r/(?:[^/ ]|\S)+/comments/|)([a-z0-9]{5,8})/?(?:[ .]+|\z)`)

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

	/*
	go func() {
		for {
			duration := time.Duration(rand.Int63n(2*96)) * time.Hour
			log.Println("Waiting ", duration)
			time.Sleep(duration)
			bastilleEvent <- plug.IRCMessage{"#reddit-anime", "", "", time.Now()}
		}
	}()
	go bastille(bastilleEvent, writeMessage)

	go amiami(amiAmiEvent, writeMessage)
	//go anidb(anidbEvent, writeMessage)
	go gelbooru(gelbooruEvent, writeMessage)
	go malSearch(animeEvent, "anime", writeMessage, matchMALAnime)
	go malSearch(mangaEvent, "manga", writeMessage, matchMALManga)
	go reddit(redditEvent, writeMessage)
	go help(helpEvent, writeMessage, conf.Channels)
	*/

	var plugins []plug.Plugin
	plugins = append(plugins, plug.Youtube{})

	for _, plugin := range plugins {
		plugin.Setup()
		fmt.Println(plugin)
		go scrapeAndSend(plugin.Event(), plugin.FindUri, plugin.Write, writeMessage)
	}

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
		prepared.When = time.Now()

		for _, plugin := range plugins {
			fmt.Println(plugin)
			re := *plugin.Match()
			if re.MatchString(prepared.Msg) {
				plugin.Event() <- *prepared
			}
		}
		/*
		switch {
		case matchAmiAmi.MatchString(prepared.Msg):
			amiAmiEvent <- *prepared
		case matchGelbooru.MatchString(prepared.Msg):
			//gelbooruEvent <- matchGelbooru.FindAllStringSubmatch(prepared.Msg, -1)[0][1]
		case matchMALAnime.MatchString(prepared.Msg):
			animeEvent <- *prepared
		case matchMALManga.MatchString(prepared.Msg):
			mangaEvent <- *prepared
		case matchReddit.MatchString(prepared.Msg):
			_, notFound := getFirstMatch(matchSpoilers, &prepared.Msg)
			if notFound != nil {
				redditEvent <- *prepared
			}
		//case matchYouTube.MatchString(prepared.Msg):
			//_, notFound := getFirstMatch(matchSpoilers, &prepared.Msg)
			//if notFound != nil {
				//youtubeEvent <- *prepared
			//}
		case matchHelp.MatchString(prepared.Msg):
			helpEvent <- *prepared
		//case matchAniDBSearch.MatchString(prepared.Msg):
		//anidbEvent <- prepared
		default:
		}
		*/
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

func bastille(event chan plug.IRCMessage, writeMessage chan plug.IRCMessage) {
	msgs := []string{
		"Bastille, yo brodudedudebro!!!!1",
		"Bastille, wat up homie",
		"Bastille, word",
		"Bastille, duuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuude",
		"'sup Bastille?",
	}

	for msg := range event {
		writeMessage <- plug.IRCMessage{msg.Channel, msgs[rand.Intn(len(msgs))-1], msg.User, msg.When}
	}
}

func scrapeAndSend(event chan plug.IRCMessage, findUri UriFunc, write WriteFunc, writeMessage chan plug.IRCMessage) {
	var f = func(msg plug.IRCMessage) {
		uri, err := findUri(&msg.Msg)
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

		outMsg, err := write(&msg, &body)
		if err != nil {
			log.Println(err)
			return
		}

		writeMessage <- *outMsg
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

/*
func amiami(event chan plug.IRCMessage, writeMessage chan plug.IRCMessage) {
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
		func(msg *plug.IRCMessage, body *string) error {
			title, err := getFirstMatch(matchTitle, body)
			if err != nil {
				return err
			}

			writeMessage <- plug.IRCMessage{msg.Channel, "[AmiAmi] " + matchDiscount.ReplaceAllLiteralString(*title, ""), msg.User, msg.When}
			return nil
		})
}

func reddit(event chan plug.IRCMessage, writeMessage chan plug.IRCMessage) {
	matchTitle := regexp.MustCompile(`.*<title>(.+)</title>.*`)

	scrapeAndSend(event, func(msg *string) (*string, error) {
		uri, err := getFirstMatch(matchReddit, msg)
		if err != nil {
			return nil, err
		}

		fullUri := "http://reddit.com/" + *uri
		return &fullUri, nil
	},
		func(msg *plug.IRCMessage, body *string) error {
			title, err := getFirstMatch(matchTitle, body)
			if err != nil {
				return err
			}

			cleanTitle := html.UnescapeString(*title)
			if cleanTitle != "reddit.com: page not found" {
				_, notFound := getFirstMatch(matchSpoilers, &cleanTitle)
				if notFound != nil {
					writeMessage <- plug.IRCMessage{msg.Channel, "[Reddit] " + cleanTitle, msg.User, msg.When}
				}
			} else {
				return errors.New("Page not found")
			}
			return nil
		})
}

func gelbooru(event chan plug.IRCMessage, writeMessage chan plug.IRCMessage) {
	type Post struct {
		post string
		tags string `xml:",attr"`
	}

	for msg := range event {
		resp, err := http.Get("http://gelbooru.com/index.php?page=dapi&s=post&q=index&tags&id=" + msg.Msg)
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
		writeMessage <- plug.IRCMessage{msg.Channel, "tobedone", msg.User, msg.When}
	}
}

func anidb(event chan plug.IRCMessage, writeMessage chan plug.IRCMessage) {
	cache := make(map[string]string)

	for msg := range event {
		val, ok := cache[msg.Msg]
		if ok {
			writeMessage <- plug.IRCMessage{msg.Channel, val, msg.User, msg.When}
		} else {
			// totally broken :(
			resp, err := http.Get("http://anisearch.outrance.pl/index.php?task=search&query=" + msg.Msg)
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

			//cache[msg.Msg] = ""
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

func malSearch(event chan plug.IRCMessage, searchType string, writeMessage chan plug.IRCMessage, match *regexp.Regexp) {
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
		func(msg *plug.IRCMessage, body *string) error {
			if len(*body) < 10 {
				writeMessage <- plug.IRCMessage{msg.Channel, "┐('～`；)┌", msg.User, msg.When}
				return errors.New("No results")
			}

			var r Results
			err := json.Unmarshal([]byte(*body), &r)
			if err != nil {
				writeMessage <- plug.IRCMessage{msg.Channel, "┐('～`；)┌", msg.User, msg.When}
				return err
			}
			fmt.Printf("%v\n", r)

			var results = ""
			var nsfw = false
			reference, _ := getFirstMatch(match, &msg.Msg)

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

			writeMessage <- plug.IRCMessage{msg.Channel, results, msg.User, msg.When}
			return nil
		})
}

func help(event chan plug.IRCMessage, writeMessage chan plug.IRCMessage, excludedChannels []string) {
	for msg := range event {
		stop := false
		for _, channel := range excludedChannels {
			if msg.Channel == channel {
				stop = true
				break
			}
		}
		if stop {
			continue
		}
		helpMsg := plug.IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When}
		terms, err := getFirstMatch(matchHelpTerms, &msg.Msg)
		if err != nil {
			helpMsg.Msg = "Usage: help mal"
			writeMessage <- helpMsg
			continue
		}
		switch {
		case *terms == "mal":
			helpMsg.Msg = "MAL search: !anime lain | !manga monster"
		}
		writeMessage <- helpMsg
	}
}
*/
