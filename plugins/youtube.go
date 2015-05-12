package plugins

import (
	"errors"
	"encoding/json"
	"log"
	"net/url"
	"path"
	"regexp"
)

type Youtube struct {
	plugin
	spoiler, title, user *regexp.Regexp
	key string
}

type YoutubeConf struct {
	Key string `json:"key"`
}

func (plug *Youtube) Setup(write chan IRCMessage, conf PluginConf) {
	plug.write = write
	plug.key = conf.Youtube.Key
	plug.match = regexp.MustCompile(`((?:https?://|)(?:www\.|m\.|)(?:youtu(?:\.be|be\.com)(?:/v/|/watch\?v=|/)[^\s/]+))(?: |$)`)
	plug.spoiler = regexp.MustCompile(`(?i)(.*spoil.*)`)
	plug.title = regexp.MustCompile(`<title>(.+) - YouTube</title>`)
	plug.user = regexp.MustCompile(`<div[^>]+class="[^">]*yt-user-info[^">]*"[^>]*><a[^>]*>([^<]+)</a>`)
	plug.event = make(chan IRCMessage, 1000)
	scrapeAndSend(plug)
	return
}

func (plug *Youtube) FindUri(candidate *string) (uri *string, err error) {
	uri, err = GetFirstMatch(plug.match, candidate)
	if err != nil {
		return
	}

	parsed, err := url.Parse(*uri)
	if err != nil {
		return
	}

	if parsed.Path == *uri {
		newuri := "http://" + *uri
		parsed, err = url.Parse(newuri)
		if err != nil {
			return
		}
	}

	full_start := "https://www.googleapis.com/youtube/v3/videos?key=" + plug.key + "&part=snippet&id="

	if ok, _ := path.Match("/v/*", parsed.Path); ok {
		_, file := path.Split(parsed.Path)
		full := full_start + file
		uri = &full
	} else if ok, _ = path.Match("/watch", parsed.Path); ok {
		query := parsed.Query()

		val, ok := query["v"]
		if !ok || len(val) < 1 {
			err = errors.New("Could not find video id")
			return
		}

		full := full_start + val[0]
		uri = &full
	} else if ok, _ = path.Match("/*", parsed.Path); ok {
		// This condition must come last because it will match those above it as well
		_, file := path.Split(parsed.Path)
		full := full_start + file
		uri = &full
	} else {
		err = errors.New("Could not find URI")
	}

	return
}

func (plug Youtube) Write(msg *IRCMessage, body *string) (err error) {
	type snippet struct {
		Title string `json:"title"`
		ChannelTitle string `json:"channelTitle"`
	}

	type item struct {
		Snippet snippet `json:"snippet"`
	}

	type results struct {
		Item []item `json:"items"`
	}

	var dat results
	err = json.Unmarshal([]byte(*body), &dat)
	if err != nil {
		return
	}

	if len(dat.Item) == 0 {
		log.Println("Error:" + *body)
		return
	}

	title := dat.Item[0].Snippet.Title
	user := dat.Item[0].Snippet.ChannelTitle
	_, notFound := GetFirstMatch(plug.spoiler, &title)
	if notFound != nil {
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
		  Msg: "[YouTube] " + title + " uploaded by " + user}
			//Msg: "[YouTube] " + html.UnescapeString(*title+" uploaded by "+*user)}
	} else {
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
			Msg: "[YouTube] [[Title omitted due to possible spoilers]] uploaded by " + user}
	}

	return
}

func (plug Youtube) Match(msg *IRCMessage) bool {
	return plug.match.MatchString(msg.Msg)
}

func (plug Youtube) Event() chan IRCMessage {
	return plug.event
}
