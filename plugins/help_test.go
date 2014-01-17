package plugins

import (
	"testing"
)

func TestHelpMatch(t *testing.T) {
	plug := Help{}
	plug.Setup(make(chan IRCMessage), PluginConf{})

	tests := []struct {
		msg     string
		matched bool
	}{
		{"Laala, would you please help me", true},
		{"Laala, please help me", true},
		{"Laala, help me", true},
		{"Laala: help me", true},
		{"Laala~ help me", true},
		{"Laala help me", true},

		{"Laala, help me!", true},
		{"Laala, help me?", true},
		{"Laala, help me.", true},
		{"Laala, help me,", true},
		{"Laala, help me~", true},
		{"Laala, help me~!.,-", true},

		{"Laala, me help", false},
		{"Laala, you please me help would", false},
		{"Laala, you please me help", false},
		{"Laala, you me help", false},
		{"Laala, I hate you", false},

		{"Laala, tell me about yourself", true},
		{"Laala, please, tell me about yourself", true},
		{"Laala, tell me about yourself, please", true},
		{"Laala, please, tell me about yourself, please", true},

		{"Laala, how do I search for anime", true},
		{"Laala, how do I search for anime???? ", true},
		{"Laala, how do I search for manga", true},
		{"Laala, how do I search for manga?!", true},
	}
	for _, test := range tests {
		result := plug.match.MatchString(test.msg)
		if result != test.matched {
			t.Error(test.msg, "expected", test.matched, "but got", result)
		}
	}
}
