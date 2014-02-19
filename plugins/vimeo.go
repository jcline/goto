package plugins

import (
	"log"
	"encoding/json"
	"html"
	"regexp"
	"errors"
	"strings"
)

var VimeoNoResultsError = errors.New("Vimeo: No results")
var VimeoEmptyResultsError = errors.New("Vimeo: Some of the fields were empty :(")

type resultWrapper struct {
	Results []jsonResult
}

type jsonResult struct {
	User string `json:"user_name"`
	Title string `json:"title"`
}

type Vimeo struct {
	plugin
	spoiler *regexp.Regexp
}

func (plug *Vimeo) Setup(write chan IRCMessage, conf PluginConf) {
	plug.write = write
	plug.match = regexp.MustCompile(`(?:https?://|)(?:www\.|)(?:vimeo.com/)(\S+)`)
	plug.spoiler = regexp.MustCompile(`(?i)(.*spoil.*)`)
	plug.event = make(chan IRCMessage, 1000)
	scrapeAndSend(plug)
	return
}

func (plug *Vimeo) FindUri(candidate *string) (uri *string, err error) {
	uri, err = GetFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}
	full := "http://vimeo.com/api/v2/video/" + *uri + ".json"
	uri = &full
	return
}

func (plug Vimeo) Write(msg *IRCMessage, body *string) (err error) {
	var result []jsonResult
	log.Println(*body)
	err = json.Unmarshal([]byte(*body), &result)
	if err != nil {
		return
	}

	if len(result) != 1 {
		err = VimeoNoResultsError
		return
	}

	title := strings.TrimSpace(result[0].Title)
	user := strings.TrimSpace(result[0].User)
	if title == "" || user == "" {
		err = VimeoEmptyResultsError
		return
	}

	_, notFound := GetFirstMatch(plug.spoiler, &title)
	if notFound != nil {
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
			Msg: "[Vimeo] " + html.UnescapeString(title+" uploaded by "+user)}
	} else {
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
			Msg: "[Vimeo] [[Title omitted due to possible spoilers]] uploaded by " + user}
	}

	return
}

func (plug Vimeo) Match() *regexp.Regexp {
	return plug.match
}

func (plug Vimeo) Event() chan IRCMessage {
	return plug.event
}
