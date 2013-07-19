package tldrtweet

import (
	"container/list"
	"fmt"
	"github.com/jzelinskie/reddit"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type tldrItem struct {
	Content string
	Author  string
	Created float64
}

var subReddits = []string{
	"askreddit",
	"funny",
	"videos",
	"games",
	"iama",
	"aww",
	"worldnews",
	"geek",
	"nosleep",
	"programming",
	"pics",
	"gaming",
	"technology",
	"cyberpunk",
	"science",
	"woahdude",
}

const (
	// Maximum size of a tweet
	tweetSize = 140
	// 24 Tweets Per Day * 7 Days A Week = 168 Tweets A Week
	maxNumberOfTweets = 168
)

var commentSet = map[string]bool{}
var subRedditIndex = 0

func RunBot() {
	resetCommentSet()
	for {
		success := crawlAndTweet(subReddits[subRedditIndex])
		subRedditIndex = (subRedditIndex + 1) % len(subReddits)
		if success {
			break
		}
	}
}

func resetCommentSet() {
	if len(commentSet) >= maxNumberOfTweets {
		// Let the garbage collector take care of getting rid of the contents of the map
		// by assigning the comment set to a new map
		commentSet = make(map[string]bool)
	}
}

func crawlAndTweet(subReddit string) bool {
	success := false
	posts, err := reddit.SubredditHeadlines(subReddit)
	fmt.Printf("Crawling /r/%s\n", subReddit)
	if noError(err) {
		tldrItemList := list.New()
		for _, post := range posts {
			// Sleep for 2 seconds as a niave way to keep the number of hits down to a max of 30 a min
			time.Sleep(2 * time.Second)
			comments, err := reddit.GetComments(post)
			if noError(err) {
				processComments(comments, tldrItemList)
			}
		}
		success = tryTweetItems(tldrItemList)
	}
	return success
}

func tryTweetItems(list *list.List) bool {
	success := false
	if list.Len() > 0 {
		for tweetItem := list.Front(); tweetItem != nil; tweetItem = tweetItem.Next() {
			success = tryTweetComment(tweetItem.Value.(tldrItem).Content)
			if success {
				break
			}
		}
	}
	return success
}

func tryTweetComment(message string) bool {
	success := false
	if tryAddComment(message) {
		fmt.Printf("Tweet: %s\n", message)
		client, err := logIn()
		if noError(err) {
			err = tweetMessage(message, client)
			if noError(err) {
				success = true
			}

		}
	}
	return success
}

func tryAddComment(comment string) bool {
	if commentSet[comment] {
		return false
	}
	commentSet[comment] = true
	return true
}

func processComments(comments reddit.Comments, list *list.List) {
	// Simple search and print of tldr comments
	for _, comment := range comments {
		formattedBody := strings.ToLower(comment.Body)
		found, sentence := extractTLDR(formattedBody)
		// Only add tweetable items to the list of candidates, i.e. less than 140 characters
		if found && len(sentence) < tweetSize {
			foundItem := tldrItem{Content: sentence, Author: comment.Author, Created: comment.Created}
			fmt.Printf("%v\n", foundItem)
			list.PushBack(foundItem)
		}
	}
}

func extractTLDR(body string) (bool, string) {
	bodyContent := strings.Fields(body)
	// If tl;dr or tl dr exists within the body of the string
	// stop at that index and extract out until either the end of
	// the string or the first period found
	for i := 0; i < len(bodyContent); i++ {
		if bodyContent[i] == "tl;dr" || bodyContent[i] == "tldr" {
			tldrSentence := bodyContent[i]
			// Only continue if there is data after the tldr string
			if i < len(bodyContent)-1 {
				for j := i + 1; j < len(bodyContent); j++ {
					tldrSentence = fmt.Sprintf("%s %s", tldrSentence, bodyContent[j])
					if strings.Contains(bodyContent[j], ".") {
						return true, tldrSentence
					}
				}
				return true, tldrSentence
			}
		}
	}
	return false, ""
}

// The loading of credentials, the login, and tweeting functionality
// has been addapted from the example provided with github.com/kurrik/twittergo
func logIn() (client *twittergo.Client, err error) {
	credentials, err := ioutil.ReadFile("CREDENTIALS")
	if noError(err) {
		lines := strings.Split(string(credentials), "\n")
		config := &oauth1a.ClientConfig{
			ConsumerKey:    lines[0],
			ConsumerSecret: lines[1],
		}
		user := oauth1a.NewAuthorizedConfig(lines[2], lines[3])
		client = twittergo.NewClient(config, user)
	}
	return
}

func tweetMessage(message string, client *twittergo.Client) error {
	data := url.Values{}
	data.Set("status", message)
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("POST", "/1.1/statuses/update.json", body)
	if noError(err) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err = client.SendRequest(req)
	}
	return err
}

func noError(err error) bool {
	if err != nil {
		fmt.Printf("%v", err)
		return false
	}
	return true
}
