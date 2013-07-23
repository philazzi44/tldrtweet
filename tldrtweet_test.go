package tldrtweet

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestBot(t *testing.T) {
	fileData, err := ioutil.ReadFile("CREDENTIALS.txt")
	if err != nil {
		fmt.Println(err)
	} else {
		credentials := string(fileData)
		bot := New()
		bot.InitializeBot(credentials)
		bot.RunBotCrawl()
		bot.RunBotTweet()
	}
}
