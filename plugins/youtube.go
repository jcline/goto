package plugins

import (
	"errors"
	"html"
	"net/url"
	"path"
	"regexp"
)

type Youtube struct {
	plugin
	spoiler, title, user *regexp.Regexp
}

func (plug *Youtube) Setup(write chan IRCMessage, conf PluginConf) {
	plug.write = write
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

	if ok, _ := path.Match("/v/*", parsed.Path); ok {
		_, file := path.Split(parsed.Path)
		full := "http://www.youtube.com/watch?v=" + file
		uri = &full
	} else if ok, _ = path.Match("/watch", parsed.Path); ok {
		query := parsed.Query()

		val, ok := query["v"]
		if !ok || len(val) < 1 {
			err = errors.New("Could not find video id")
			return
		}

		full := "http://www.youtube.com/watch?v=" + val[0]
		uri = &full
	} else if ok, _ = path.Match("/*", parsed.Path); ok {
		// This condition must come last because it will match those above it as well
		_, file := path.Split(parsed.Path)
		full := "http://www.youtube.com/watch?v=" + file
		uri = &full
	} else {
		err = errors.New("Could not find URI")
	}

	return
}

func (plug Youtube) Write(msg *IRCMessage, body *string) (err error) {
	title, err := GetFirstMatch(plug.title, body)
	if err != nil {
		return
	}

	user, err := GetFirstMatch(plug.user, body)
	if err != nil {
		return
	}

	_, notFound := GetFirstMatch(plug.spoiler, title)
	if notFound != nil {
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
			Msg: "[YouTube] " + html.UnescapeString(*title+" uploaded by "+*user)}
	} else {
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
			Msg: "[YouTube] [[Title omitted due to possible spoilers]] uploaded by " + *user}
	}

	return
}

func (plug Youtube) Match(msg *IRCMessage) bool {
	return plug.match.MatchString(msg.Msg)
}

func (plug Youtube) Event() chan IRCMessage {
	return plug.event
}
