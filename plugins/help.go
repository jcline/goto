package plugins

import (
	"log"
	"regexp"
	"strings"
)

type Help struct {
	plugin
	help map[string]string
}

func (plug *Help) Setup(write chan IRCMessage, conf PluginConf) {
	plug.write = write
	plug.match = regexp.MustCompile(`^help(.*)`)
	plug.event = make(chan IRCMessage, 1000)

	// TODO: Provide a way to specify bot names at this level
	regexPrefix := `(?:^Laala[,:~]{0,1} `
	regexPostfix := `)`
	regexes := []string{
		regexPrefix + `(?:please[,] ){0,1}tell me (?P<command>about) yourself(?:[,] please){0,1}` + regexPostfix,
		regexPrefix + `(?:please |would you please ){0,1}(?P<command>help) me` + regexPostfix,
		regexPrefix + `how do I search for (?P<command>anime|manga)` + regexPostfix,
		//regexPrefix + `` + regexPostfix,
	}

	plug.match = regexp.MustCompile(`(?i:` + strings.Join(regexes, `|`) + `)`)

	plug.help = map[string]string{
		"about": "No! I want to leave this planet!",
		"help":  "Please ask me what you would like to know~",
		"anime": "!anime Galaxy Express 999",
		"manga": "!manga Galaxy Express 999",
	}
	go plug.Action()
	return
}

func (plug Help) Action() {
	for msg := range plug.event {
		key := ""
		query, err := getMatch(plug.match, &msg.Msg)
		if err == nil {
			for index, val := range query[0] {
				if index == 0 {
					continue
				}

				if val != "" && plug.match.SubexpNames()[index] == "command" {
					key = val
				}
			}

			if val, ok := plug.help[key]; ok {
				plug.write <- IRCMessage{Channel: msg.User, Msg: val, User: msg.User, When: msg.When}
			} else {
				plug.write <- IRCMessage{Channel: msg.User, Msg: "┐('～`；)┌", User: msg.User, When: msg.When}
			}
		} else {
			log.Println(err)
		}
	}
}

func (plug Help) Match(msg *IRCMessage) bool {
	return plug.match.MatchString(msg.Msg)
}

func (plug Help) Event() chan IRCMessage {
	return plug.event
}
