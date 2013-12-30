package plugins

import (
	"html"
	"regexp"
)

type MyFigureCollection struct {
	plugin
	spoiler, title, user *regexp.Regexp
}

func (plug *MyFigureCollection) Setup(write chan IRCMessage) {
	plug.write = write
	plug.match = regexp.MustCompile(`(?:https?://|)(?:www\.|)(myfigurecollection.net/item/\S+)`)
	plug.title = regexp.MustCompile(`.*<meta name="description" content="([^"]+)".*/>.*`)
	plug.event = make(chan IRCMessage, 1000)
	scrapeAndSend(plug)
	return
}

func (plug *MyFigureCollection) FindUri(candidate *string) (uri *string, err error) {
	uri, err = GetFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}
	full := "http://" + *uri
	uri = &full
	return
}

func (plug MyFigureCollection) Write(msg *IRCMessage, body *string) (err error) {
	title, err := GetFirstMatch(plug.title, body)
	if err != nil {
		return
	}

	plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
		Msg: "[MFC] " + html.UnescapeString(*title)}

	return
}

func (plug MyFigureCollection) Match() *regexp.Regexp {
	return plug.match
}

func (plug MyFigureCollection) Event() chan IRCMessage {
	return plug.event
}

