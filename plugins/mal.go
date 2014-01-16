package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jcline/DamerauLevenshteinDistance"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type results []result
type result struct {
	distance, Id                  int
	Classification, Title, search string
	computed                      bool
}

func (r results) Len() int {
	return len(r)
}

func (r results) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r results) Less(i, j int) bool {
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

	full := "http://mal-api.com/" + *plug.searchType + "/search?q=" + url.QueryEscape(*terms)
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

	var r results
	err = json.Unmarshal([]byte(*body), &r)
	if err != nil {
		plug.write <- IRCMessage{Channel: msg.Channel, Msg: "┐('～`；)┌", User: msg.User, When: msg.When}
		return
	}
	fmt.Printf("%v\n", r)

	var resultString = ""
	var nsfw = false
	reference, _ := GetFirstMatch(plug.match, &msg.Msg)

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
		if *plug.searchType == "anime" {
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

			resultString += result.Title + class + " http://myanimelist.net/" + *plug.searchType + "/" + strconv.Itoa(result.Id) + "  "
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

	if len(r) > 3 {
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
