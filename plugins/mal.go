package plugins

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/jcline/DamerauLevenshteinDistance"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	//"strings"
)

type entries []entry
type result struct {
	Entries entries `xml:"entry"`
}

type entry struct {
	Title    string `xml:"title"`
	Id       int    `xml:"id"`
	distance int
	search   string
	computed bool
}

type MalConf struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

func (r entries) Len() int {
	return len(r)
}

func (r entries) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r entries) Less(i, j int) bool {
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

type Mal struct {
	plugin
	spoiler, title, typeMatch *regexp.Regexp
	searchType, terms         *string
}

func (plug *Mal) Setup(write chan IRCMessage, conf PluginConf) {
	plug.write = write
	plug.match = regexp.MustCompile(`^!(?:anime|manga) (.+)`)
	plug.spoiler = regexp.MustCompile(`(?i)(.*spoil.*)`)
	plug.title = regexp.MustCompile(`.*<title>(.+)</title>.*`)
	plug.typeMatch = regexp.MustCompile(`^!(anime|manga) .+`)
	plug.event = make(chan IRCMessage, 1000)

	malScrapeAndSend(plug, conf.Mal.User, conf.Mal.Password)
	return
}

func (plug *Mal) FindUri(candidate *string) (uri *string, err error) {
	terms, err := GetFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}

	plug.searchType, err = GetFirstMatch(plug.typeMatch, candidate)
	if err != nil {
		uri = nil
		return
	}

	full := "http://myanimelist.net/api/" + *plug.searchType + "/search.xml?q=" + url.QueryEscape(*terms)
	plug.terms = terms
	uri = &full
	fmt.Println(plug)
	return
}

func (plug Mal) Write(msg *IRCMessage, body *string) (err error) {
	fmt.Println(plug)
	if len(*body) < 10 {
		plug.write <- IRCMessage{Channel: msg.Channel, Msg: "┐('～`；)┌", User: msg.User, When: msg.When}
		err = errors.New("No results")
		return
	}

	fmt.Printf("%v\n", *body)
	unescaped := html.UnescapeString(*body)
	//fmt.Printf("%v\n", unescaped)
	var r result
	err = xml.Unmarshal([]byte(unescaped), &r)
	if err != nil {
		plug.write <- IRCMessage{Channel: msg.Channel, Msg: "┐('～`；)┌", User: msg.User, When: msg.When}
		return
	}
	fmt.Printf("%v\n", r)

	var resultString = ""
	var nsfw = false
	reference, _ := GetFirstMatch(plug.match, &msg.Msg)

	for _, e := range r.Entries {
		//r[i].Title = html.UnescapeString(r[i].Title)
		e.search = *reference
		e.computed = false
	}
	sort.Sort(r.Entries)

	length := 2
	if len(r.Entries) < length {
		length = len(r.Entries)
	}
	for count, result := range r.Entries {
		if *plug.searchType == "anime" {
			/*
				class := result.Classification
				if class != "" {
					if strings.Contains(class, "Rx") ||
						strings.Contains(class, "R+") ||
						strings.Contains(class, "Hentai") {
						nsfw = true
					}
					class = " [Rating " + class + "]"
					nsfw = true
				} else {
					nsfw = true
				}
			*/
			nsfw = true

			resultString += result.Title + /*class + */ " http://myanimelist.net/" + *plug.searchType + "/" + strconv.Itoa(result.Id) + "  "
		} else {
			resultString += result.Title + " http://myanimelist.net/" + *plug.searchType + "/" + strconv.Itoa(result.Id) + "  "
			nsfw = true
		}
		if count >= length {
			break
		}
	}

	if nsfw {
		resultString = "NSFW " + resultString
	}

	if len(r.Entries) > 3 {
		resultString += "More: " + "http://myanimelist.net/" + *plug.searchType + ".php?q=" + url.QueryEscape(*plug.terms)
	}

	plug.write <- IRCMessage{Channel: msg.Channel, Msg: resultString, User: msg.User, When: msg.When}
	return
}

func (plug Mal) Match() *regexp.Regexp {
	return plug.match
}

func (plug Mal) Event() chan IRCMessage {
	return plug.event
}

func malScrapeAndSend(plug scrapePlugin, user string, password string) {
	var f = func(msg IRCMessage) {
		uri, err := plug.FindUri(&msg.Msg)
		if err != nil {
			log.Println(err)
			return
		}

		client := &http.Client{}
		request, err := http.NewRequest("GET", *uri, nil)
		request.SetBasicAuth(user, password)
		resp, err := client.Do(request)
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

		err = plug.Write(&msg, &body)
		if err != nil {
			log.Println(err)
			return
		}
	}

	go func() {
		for msg := range plug.Event() {
			go f(msg)
		}
	}()
}
