package plugins

import (
	"html"
	"regexp"
)

type Plamoya struct {
	plugin
	spoiler, title, user *regexp.Regexp
}

func (plug *Plamoya) Setup(write chan IRCMessage) {
	plug.write = write
	plug.match = regexp.MustCompile(`(?:https?://|)(?:www\.|)(plamoya.com/\S+)`)
	plug.title = regexp.MustCompile(`.*<title>([^<]+) \([^)<:]+\) [^:<]+ : PLAMOYA</title>.*`)
	plug.event = make(chan IRCMessage, 1000)
	scrapeAndSend(plug)
	return
}

func (plug *Plamoya) FindUri(candidate *string) (uri *string, err error) {
	uri, err = getFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}
	full := "http://" + *uri
	uri = &full
	return
}

func (plug Plamoya) Write(msg *IRCMessage, body *string) (err error) {
	title, err := getFirstMatch(plug.title, body)
	if err != nil {
		return
	}

	plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
		Msg: "[Plamoya] " + html.UnescapeString(*title)}

	return
}

func (plug Plamoya) Match() *regexp.Regexp {
	return plug.match
}

func (plug Plamoya) Event() chan IRCMessage {
	return plug.event
}

