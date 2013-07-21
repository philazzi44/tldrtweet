package tldrtweet

import (
	"container/list"
	"errors"
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

//////////////////////////////////////////////////////////////////////////////////////////////////
// Definitions                                                                                 //
////////////////////////////////////////////////////////////////////////////////////////////////

type tldrItem struct {
	Content string
	Author  string
	Created float64
}

type TweetBot struct {
	commentSet       map[string]bool
	commentList      *list.List
	subRedditList    *list.List
	currentSubReddit *list.Element
	credentials      string
}

const (
	// Maximum size of a tweet
	tweetSize = 140
	// 24 Tweets Per Day * 7 Days A Week = 168 Tweets A Week
	maxNumberOfTweets = 168
)

const (
	CommentsSaveFile = "comments.dat"
	SubRedditFile    = "subreddits.dat"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// Bot Initalization and Loading/Saving                                                        //
////////////////////////////////////////////////////////////////////////////////////////////////
func New() *TweetBot {
	return &TweetBot{
		commentSet:    make(map[string]bool),
		commentList:   list.New(),
		subRedditList: list.New()}
}

func (bot *TweetBot) InitializeBot(credentials string) {
	bot.setBotTwitterCredentials(credentials)
	bot.loadCommentFromFile()
	err := bot.loadSubreddits()
	if err != nil {
		panic(err)
	}
}

func (bot *TweetBot) setBotTwitterCredentials(credentials string) {
	bot.credentials = credentials
}

func (bot *TweetBot) loadCommentFromFile() {
	fileLines, err := loadLines(CommentsSaveFile)
	if err == nil {
		for i := 0; i < len(fileLines); i++ {
			tryAddComment(fileLines[i], bot)
		}
	}
}

func (bot *TweetBot) saveCommentToFile() {
	fileSaveData := ""
	for comment := bot.commentList.Front(); comment != nil; comment = comment.Next() {
		fileSaveData = fmt.Sprintf("%s%s\n", fileSaveData, comment.Value.(string))
	}
	err := ioutil.WriteFile(CommentsSaveFile, []byte(fileSaveData), 0644)
	if !noError(err) {
		fmt.Println("Failed save! Please see error above.")
	}
}

func (bot *TweetBot) resetCommentSet() {
	if bot.commentList.Len() >= maxNumberOfTweets {
		// Remove the 84 oldest tweets
		for {
			if bot.commentList.Len() > (maxNumberOfTweets / 2) {
				commentListItem := bot.commentList.Back()
				comment := commentListItem.Value.(string)
				// Remove from both the set and the list
				delete(bot.commentSet, comment)
				bot.commentList.Remove(commentListItem)
			} else {
				break
			}
		}
	}
}

func (bot *TweetBot) loadSubreddits() error {
	fileLines, err := loadLines(SubRedditFile)
	if err == nil {
		for i := 0; i < len(fileLines); i++ {
			bot.subRedditList.PushFront(fileLines[i])
		}
	}

	if bot.subRedditList.Len() < 1 {
		return errors.New("Failed to load any subreddits!")
	}
	return nil
}

func (bot *TweetBot) getSubReddit() string {

	// If the current subreddit is empty, go back to the
	// start of the list
	if bot.currentSubReddit == nil {
		bot.currentSubReddit = bot.subRedditList.Front()
	}

	subRedditName := bot.currentSubReddit.Value.(string)

	// After the subreddit's name is retrieved, move onto the next
	if bot.currentSubReddit != nil {
		bot.currentSubReddit = bot.currentSubReddit.Next()
	}

	return subRedditName
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Processing and Tweeting                                                                     //
////////////////////////////////////////////////////////////////////////////////////////////////
func (bot *TweetBot) RunBot() {
	if len(bot.credentials) < 1 {
		fmt.Println("No twitter credentials specified!")
	} else {
		bot.resetCommentSet()
		for {
			success := crawlAndTweet(bot.getSubReddit(), bot)
			if success {
				// On a successful run save the updated tweets
				bot.saveCommentToFile()
				break
			}
		}
	}
}

func tryAddComment(comment string, bot *TweetBot) bool {
	if bot.commentSet[comment] {
		return false
	}
	bot.commentSet[comment] = true
	bot.commentList.PushFront(comment)
	return true
}

func crawlAndTweet(subReddit string, bot *TweetBot) bool {
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
		success = tryTweetItems(tldrItemList, bot)
	}
	return success
}

func tryTweetItems(list *list.List, bot *TweetBot) bool {
	success := false
	if list.Len() > 0 {
		for tweetItem := list.Front(); tweetItem != nil; tweetItem = tweetItem.Next() {
			success = tryTweetComment(tweetItem.Value.(tldrItem).Content, bot)
			if success {
				break
			}
		}
	}
	return success
}

func tryTweetComment(message string, bot *TweetBot) bool {
	success := false
	if tryAddComment(message, bot) {
		fmt.Printf("Tweet: %s\n", message)
		client, err := logIn(bot)
		if noError(err) {
			err = tweetMessage(message, client)
			if noError(err) {
				success = true
			}
		}
	}
	return success
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

//////////////////////////////////////////////////////////////////////////////////////////////////
// Twitter Specific                                                                            //
////////////////////////////////////////////////////////////////////////////////////////////////

// The loading of credentials, the login, and tweeting functionality
// has been addapted from the example provided with github.com/kurrik/twittergo
func logIn(bot *TweetBot) (client *twittergo.Client, err error) {
	if len(bot.credentials) > 0 {
		lines := strings.Split(string(bot.credentials), "\n")
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

//////////////////////////////////////////////////////////////////////////////////////////////////
// Utility Methods                                                                             //
////////////////////////////////////////////////////////////////////////////////////////////////

func noError(err error) bool {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	return true
}

func loadLines(filename string) ([]string, error) {
	fileData, err := ioutil.ReadFile(filename)
	if noError(err) {
		return strings.Split(string(fileData), "\n"), nil
	}
	return nil, err
}
