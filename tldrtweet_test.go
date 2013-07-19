package tldrtweet

import (
	"testing"
)

func TestBot(t *testing.T) {
	bot := New()
	bot.SetBotTwitterCredentialsPath("CREDENTIALS")
	bot.RunBot()
}
