package plugins

import (
	"regexp"
)

type AmiAmi struct {
	plugin
	discount, title *regexp.Regexp
}

func (plug *AmiAmi) Setup(write chan IRCMessage) {
	plug.write = write
	plug.discount = regexp.MustCompile(`[0-9]+\%OFF `)
	plug.match = regexp.MustCompile(`(?:https?://|)(?:www\.|)amiami.com/((?:[^/]|\S)+/detail/\S+)`)
	plug.title = regexp.MustCompile(`.*<meta property="og:title" content="(.+)" />.*`)
	plug.event = make(chan IRCMessage, 1000)
	scrapeAndSend(plug)
	return
}

func (plug *AmiAmi) FindUri(candidate *string) (uri *string, err error) {
	uri, err = getFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}
	full := "http://amiami.com/" + *uri
	uri = &full
	return
}

func (plug AmiAmi) Write(msg *IRCMessage, body *string) (err error) {
	title, err := getFirstMatch(plug.title, body)
	if err != nil {
		return
	}

	plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
	Msg: "[AmiAmi] " + plug.discount.ReplaceAllLiteralString(*title, "")}

	return
}

func (plug AmiAmi) Match() *regexp.Regexp {
	return plug.match
}

func (plug AmiAmi) Event() chan IRCMessage {
	return plug.event
}
