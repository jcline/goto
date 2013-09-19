package plugins

import (
	"errors"
	"html"
	"regexp"
)

type Reddit struct {
	plugin
	spoiler, title *regexp.Regexp
}

func (plug *Reddit) Setup() {
	plug.match = regexp.MustCompile(`(?:http://|)(?:www\.|https://pay\.|)redd(?:\.it|it\.com)/(?:r/(?:[^/ ]|\S)+/comments/|)([a-z0-9]{5,8})/?(?:[ .]+|\z)`)
	plug.spoiler = regexp.MustCompile(`(?i)(.*spoil.*)`)
	plug.title = regexp.MustCompile(`.*<title>(.+)</title>.*`)
	plug.event = make(chan IRCMessage, 1000)
	return
}

func (plug *Reddit) FindUri(candidate *string) (uri *string, err error) {
	uri, err = getFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}
	full := "http://reddit.com/" + *uri
	uri = &full
	return
}

func (plug Reddit) Write(msg *IRCMessage, body *string) (outMsg *IRCMessage, err error) {
	outMsg = nil
	title, err := getFirstMatch(plug.title, body)
	if err != nil {
		return
	}

	cleanTitle := html.UnescapeString(*title)
	if cleanTitle != "reddit.com: page not found" {
		_, notFound := getFirstMatch(plug.spoiler, &cleanTitle)
		if notFound != nil {
			outMsg = &IRCMessage{msg.Channel, "[Reddit] " + cleanTitle, msg.User, msg.When}
		}
	} else {
		err = errors.New("Page not found")
		return
	}

	return
}

func (plug Reddit) Match() *regexp.Regexp {
	return plug.match
}

func (plug Reddit) Event() chan IRCMessage {
	return plug.event
}
