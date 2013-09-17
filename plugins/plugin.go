package plugins

import (
	"errors"
	"regexp"
	"time"
)

type IRCMessage struct {
  Channel string
  Msg     string
  User    string
  When    time.Time
}

type Plugin interface {
  FindUri(*string) (*string, error)
  Write(*IRCMessage, *string) (*IRCMessage, error)
	Match() *regexp.Regexp
	Event() chan IRCMessage
	Setup() error
}

type plugin struct {
  match *regexp.Regexp
  event chan IRCMessage
}

func getFirstMatch(re *regexp.Regexp, matchee *string) (*string, error) {
  match := re.FindAllStringSubmatch(*matchee, -1) 
  if len(match) < 1 { 
    return nil, errors.New("Could not match")
  }
  if len(match[0]) < 2 { 
    return nil, errors.New("Could not match")
  }
  return &match[0][1], nil 
}

