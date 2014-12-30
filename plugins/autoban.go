package plugins

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"unicode"
)

type AutobanMatches struct {
	Matcher *regexp.Regexp
	Time    string   `json:"time"`
	Regex   string   `json:"match"`
	Strip   []string `json:"strip"`
	Reason  string   `json:"reason"`
}

type Autoban struct {
	plugin
	autoBans []AutobanMatches
}

type AutobanConf struct {
	Matches []AutobanMatches `json:"matchers"`
}

func (plug *Autoban) Setup(write chan IRCMessage, conf PluginConf) {
	plug.write = write
	plug.event = make(chan IRCMessage, 1000)

	var buffer bytes.Buffer
	buffer.WriteString(`(?i:`)
	// Because range copies the elements, which is totally idiotic
	matches := conf.Autoban.Matches
	for i := 0; i < len(matches); i++ {
		buffer.WriteString(matches[i].Regex)
		matches[i].Matcher = regexp.MustCompile(matches[i].Regex)
	}
	buffer.WriteString(`)`)

	plug.match = regexp.MustCompile(buffer.String())

	plug.autoBans = conf.Autoban.Matches

	go plug.Action()
	return
}

func removeNongraphic(msg string) string {
	var buffer bytes.Buffer
	for _, char := range msg {
		if unicode.IsGraphic(char) && !unicode.IsSpace(char) {
			buffer.WriteRune(char)
		}
	}
	return buffer.String()
}

func removeColors(msg string) string {
	return regexp.MustCompile("\x03.").ReplaceAllLiteralString(msg, "")
}

func (plug Autoban) Ban(msg IRCMessage) {
	time := ""
	reason := ":("
	for index, matcher := range plug.autoBans {
		cleaned := matcher.doCleanup(msg.Msg)
		if matcher.Matcher.MatchString(cleaned) {
			time = plug.autoBans[index].Time
			reason = plug.autoBans[index].Reason
		}
	}

	logMsg := fmt.Sprintf(
		"Banning user `%s` with `%s` from `%s` for `%s` at `%s`",
		msg.User,
		msg.Mask,
		msg.Channel,
		msg.Msg,
		msg.When)
	log.Println(logMsg)
	plug.write <- IRCMessage{
		Channel:   "Rodya",
		Msg:       logMsg,
		User:      msg.User,
		When:      msg.When,
		Unlimited: true,
	}

	if len(msg.Mask) < 3 {
		log.Println("msg.Mask too short to ban! %s", msg.Mask)
		return
	}

	banMsg := ""
	if time == "" {
		return
	} else {
		banMsg = fmt.Sprintf(
			"akick %s add *!*@%s !T %s %s | Laala b& '%s' for '%s'",
			msg.Channel,
			msg.Mask,
			time,
			reason,
			msg.User,
			msg.Msg)
	}

	log.Println(banMsg)
	plug.write <- IRCMessage{
		Channel:   "ChanServ",
		Msg:       banMsg,
		User:      msg.User,
		When:      msg.When,
		Unlimited: true,
	}
}

func (plug Autoban) Action() {
	for msg := range plug.event {
		go plug.Ban(msg)
	}
}

func (matcher AutobanMatches) doCleanup(msg string) string {
	cleaned := msg
	for _, stripper := range matcher.Strip {
		switch stripper {
		case "colors":
			cleaned = removeColors(cleaned)
		case "nongraphic":
			cleaned = removeNongraphic(cleaned)
		}
	}
	return cleaned
}

func (plug Autoban) Match(msg *IRCMessage) bool {
	var cleaned string
	matched := false
	for _, matcher := range plug.autoBans {
		cleaned = matcher.doCleanup(msg.Msg)
		log.Println(plug.match.String(), cleaned)
		matched = matched || plug.match.MatchString(cleaned)
		if matched {
			break
		}
	}
	log.Println(matched)

	return matched
}

func (plug Autoban) Event() chan IRCMessage {
	return plug.event
}
