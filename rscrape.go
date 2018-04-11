package rscraper

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	apiUserAgent             = "rscrape_golang_tool/v0.1-alpha"
	apiIDRegex               = "^t(1|3|5)_[A-Za-z0-9]{5,9}$"
	apiObjectTypeListing     = "Listing"
	apiObjectTypeComment     = "t1"
	apiObjectTypePost        = "t3"
	apiObjectTypeSubreddit   = "t5"
	apiObjectTypeMoreReplies = "more"

	// ListingTypeNew get newests posts in a subreddit
	ListingTypeNew = "new"

	// ListingTypeHot get hotest posts in a subreddit
	ListingTypeHot = "hot"

	// ListingTypeTop get top posts in a subreddit
	ListingTypeTop = "top"

	// ListingTopAllTime get top posts of all time in a subreddit
	ListingTopAllTime = "all"

	// ListingTopPastHour get top posts in the past hour in a subreddit
	ListingTopPastHour = "hour"

	// ListingTopPastDay get top posts in the past day in a subreddit
	ListingTopPastDay = "day"

	// ListingTopPastWeek get top posts in the past week in a subreddit
	ListingTopPastWeek = "week"

	// ListingTopPastMonth get top posts in the past month in a subreddit
	ListingTopPastMonth = "month"

	// ListingTopPastYear get top posts in the past year in a subreddit
	ListingTopPastYear = "year"
)

type apiObject struct {
	Type string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type listing struct {
	After    string      `json:"after"`
	Children []apiObject `json:"children"`
}

type moreReplies struct {
	Children []string `json:"children"`
}

// Subreddit a subreddit on reddit
type Subreddit struct {
	ID         string  `json:"id"`
	Name       string  `json:"display_name"`
	URL        string  `json:"url"`
	Title      string  `json:"title"`
	CreatedUTC float64 `json:"created_utc"`
	CreatedOn  time.Time
}

// Post a post on a subreddit
type Post struct {
	ID              string  `json:"id"`
	SubredditID     string  `json:"subreddit_id"`
	Author          string  `json:"author"`
	LinkFlairText   string  `json:"link_flair_text"`
	LinkFlairCSS    string  `json:"link_flair_css_class"`
	AuthorFlairText string  `json:"author_flair_text"`
	AuthorFlairCSS  string  `json:"author_flair_css_class"`
	Title           string  `json:"title"`
	URL             string  `json:"url"`
	PermaLink       string  `json:"permalink"`
	CreatedUTC      float64 `json:"created_utc"`
	Gilded          int     `json:"gilded"`
	Score           int     `json:"score"`
	UpVotes         int     `json:"ups"`
	DownVotes       int     `json:"downs"`
	Text            string  `json:"selftext"`
	TextHTML        string  `json:"selftext_html"`
	CreatedOn       time.Time
}

// Comment a comment on a post
type Comment struct {
	ID              string          `json:"id"`
	PostID          string          `json:"link_id"`
	ParentID        string          `json:"parent_id"`
	Author          string          `json:"author"`
	AuthorFlairText string          `json:"author_flair_text"`
	AuthorFlairCSS  string          `json:"author_flair_css_class"`
	PermaLink       string          `json:"permalink"`
	CreatedUTC      float64         `json:"created_utc"`
	Gilded          int             `json:"gilded"`
	Score           int             `json:"score"`
	UpVotes         int             `json:"ups"`
	DownVotes       int             `json:"downs"`
	Body            string          `json:"body"`
	BodyHTML        string          `json:"body_html"`
	Replies         json.RawMessage `json:"replies"`
	RepliesAfter    []string
	CreatedOn       time.Time
}

func (me *Comment) extractReplies() ([]Comment, error) {

	me.RepliesAfter = make([]string, 0)

	replies := make([]Comment, 0)

	if me.Replies == nil || len(me.Replies) < 2 || string(me.Replies) == "\"\"" {
		return replies, nil
	}

	var repliesObject apiObject

	err := json.Unmarshal(me.Replies, &repliesObject)

	if err != nil {
		return replies, err
	}

	list, err := extractListing(&repliesObject)

	if err != nil {
		return replies, err
	}

	if ok, _ := regexp.MatchString(apiIDRegex, list.After); ok {
		me.RepliesAfter = append(me.RepliesAfter, list.After)
	}

	for _, child := range list.Children {

		comment, err := extractComment(&child)

		if err != nil {

			repliesAfter, err := extractMore(&child)

			if err != nil {
				return replies, errors.New("API Object is not a Comment or More Replies")
			}

			me.RepliesAfter = append(me.RepliesAfter, repliesAfter...)
			continue
		}

		replies = append(replies, *comment)

		childReplies, err := comment.extractReplies()

		if err != nil {
			return replies, err
		}

		replies = append(replies, childReplies...)
	}

	me.Replies = nil

	return replies, nil
}

// GetSubreddit retrieve information on a specific subreddit
func GetSubreddit(subreddit string) (*Subreddit, error) {

	redditURL := getSubredditURL(subreddit)

	object, err := getResponse(redditURL.String())

	if err != nil {
		return nil, err
	}

	return extractSubreddit(object)
}

// GetPosts retrieves all posts from the specified
func GetPosts(subreddit, listingType, after, topType string) ([]Post, string, error) {

	posts := make([]Post, 0)

	redditURL := getPostsURL(subreddit, listingType, after, topType)

	object, err := getResponse(redditURL.String())

	if err != nil {
		return nil, "", err
	}

	list, err := extractListing(object)

	if err != nil {
		return nil, "", err
	}

	after = ""

	if ok, _ := regexp.MatchString(apiIDRegex, list.After); ok {
		after = list.After
	}

	for _, child := range list.Children {

		post, err := extractPost(&child)

		if err != nil {
			return nil, "", err
		}

		posts = append(posts, *post)
	}

	return posts, after, nil
}

// GetComments retrieves comments for a particular post
func GetComments(subreddit, postID, after string) ([]Comment, []string, error) {

	comments := make([]Comment, 0)

	redditURL := getCommentsURL(subreddit, postID, after)

	objects, err := getResponses(redditURL.String())

	if err != nil {
		return nil, nil, err
	}

	var list *listing

	for _, object := range objects {

		list, err = extractListing(&object)

		if err != nil {
			list = nil
			continue
		}

		if list.Children == nil || len(list.Children) == 0 {
			list = nil
			continue
		}

		_, err = extractComment(&(list.Children[0]))

		if err == nil {
			break
		} else {
			list = nil
		}
	}

	if list == nil {
		return nil, nil, errors.New("No comment listings found")
	}

	after = ""

	if ok, _ := regexp.MatchString(apiIDRegex, list.After); ok {
		after = list.After
	}

	more := make([]string, 0)

	for _, child := range list.Children {

		comment, err := extractComment(&child)

		if err != nil {

			moreComments, err := extractMore(&child)

			if err != nil {
				return nil, nil, errors.New("API Object is not a Comment or More Replies")
			}

			more = append(more, moreComments...)
			continue
		}

		comments = append(comments, *comment)

		commentReplies, err := comment.extractReplies()

		if err != nil {
			return nil, nil, err
		}

		comments = append(comments, commentReplies...)
	}

	return comments, more, nil
}

func getResponse(url string) (*apiObject, error) {

	var object apiObject

	bytes, err := get(url)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &object)

	return &object, nil
}

func getResponses(url string) ([]apiObject, error) {

	objects := make([]apiObject, 0)

	bytes, err := get(url)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &objects)

	return objects, err
}

func get(url string) ([]byte, error) {

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", apiUserAgent)

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func getSubredditURL(subreddit string) *url.URL {

	redditURL := getBaseURL()

	redditURL.Path = fmt.Sprintf("/r/%s/about.json", subreddit)

	return redditURL
}

func getPostsURL(subreddit, listingType, after, topType string) *url.URL {

	redditURL := getBaseURL()

	redditURL.Path = fmt.Sprintf("/r/%s/%s.json", subreddit, listingType)

	q := redditURL.Query()

	if ok, _ := regexp.MatchString(apiIDRegex, after); ok {
		q.Set("after", after)
	}

	if listingType == ListingTypeTop {
		switch topType {
		case ListingTopPastDay:
			q.Set("t", ListingTopPastDay)
			break
		case ListingTopPastHour:
			q.Set("t", ListingTopPastHour)
			break
		case ListingTopPastMonth:
			q.Set("t", ListingTopPastMonth)
			break
		case ListingTopPastWeek:
			q.Set("t", ListingTopPastWeek)
			break
		case ListingTopPastYear:
			q.Set("t", ListingTopPastYear)
			break
		default:
			q.Set("t", ListingTopAllTime)
		}
	}

	redditURL.RawQuery = q.Encode()

	return redditURL
}

func getCommentsURL(subreddit, postID, after string) *url.URL {

	redditURL := getBaseURL()

	if postID[0:3] == "t3_" {
		postID = postID[3:]
	}

	redditURL.Path = fmt.Sprintf("/r/%s/comments/%s.json", subreddit, postID)

	if ok, _ := regexp.MatchString(apiIDRegex, after); ok {
		q := redditURL.Query()

		q.Set("after", after)

		redditURL.RawQuery = q.Encode()
	}

	return redditURL
}

func getBaseURL() *url.URL {

	var redditURL url.URL

	redditURL.Scheme = "https"
	redditURL.Host = "reddit.com"

	return &redditURL
}

func extractListing(object *apiObject) (*listing, error) {

	if object == nil || object.Type != apiObjectTypeListing {
		return nil, errors.New("Provided API Object is not a Listing")
	}

	var result listing

	err := json.Unmarshal(object.Data, &result)

	return &result, err
}

func extractSubreddit(object *apiObject) (*Subreddit, error) {

	if object == nil || object.Type != apiObjectTypeSubreddit {
		return nil, errors.New("Provided API Object is not a Subreddit")
	}

	var result Subreddit

	err := json.Unmarshal(object.Data, &result)

	if err != nil {
		return nil, err
	}

	result.CreatedOn = time.Unix(int64(result.CreatedUTC), 0)
	return &result, err
}

func extractPost(object *apiObject) (*Post, error) {

	if object == nil || object.Type != apiObjectTypePost {
		return nil, errors.New("Provided API Object is not a Post")
	}

	var result Post

	err := json.Unmarshal(object.Data, &result)

	if err != nil {
		return nil, err
	}

	result.CreatedOn = time.Unix(int64(result.CreatedUTC), 0)
	return &result, err
}

func extractComment(object *apiObject) (*Comment, error) {

	if object == nil || object.Type != apiObjectTypeComment {
		return nil, errors.New("Provided API Object is not a Comment")
	}

	var result Comment

	err := json.Unmarshal(object.Data, &result)

	if err != nil {
		return nil, err
	}

	result.CreatedOn = time.Unix(int64(result.CreatedUTC), 0)
	return &result, err
}

func extractMore(object *apiObject) ([]string, error) {

	if object == nil || object.Type != apiObjectTypeMoreReplies {
		return nil, errors.New("Provided API Object is not More Replies")
	}

	var result moreReplies

	err := json.Unmarshal(object.Data, &result)

	if err != nil {
		return nil, err
	}

	if result.Children == nil {
		result.Children = make([]string, 0)
	}

	return result.Children, err
}
