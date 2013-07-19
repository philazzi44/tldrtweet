package tldrtweet

import (
	"testing"
)

func TestBot(t *testing.T) {
	bot := New()
<<<<<<< HEAD
	bot.SetBotTwitterCredentialsPath("CREDENTIALS.txt")
=======
	bot.SetBotTwitterCredentialsPath("CREDENTIALS")
>>>>>>> new_branch_name
	bot.RunBot()
}
