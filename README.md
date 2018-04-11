# rscraper

Golang library for scraping Reddit posts and comments

## Features

* Subreddit, Post, and Comment structs
* Simple Straight-forward API
* Does not require reddit account

## Examples

### Retrieving Subreddit data

    // Retrieve posts from the subreddit '/r/mildlyinteresting'
    if sub, err := rscraper.GetSubreddit("mildlyinteresting"); err == nil {

        fmt.Printf("the subreddit name is %s and its URL is %s\n", sub.Name, sub.URL)
    }

### Retrieving Top Posts from a subreddit

    // Grab Top posts from the past week of the /r/nfl subreddit
    posts, after, err := rscraper.GetPosts("nfl", rscraper.ListingTypeTop, "", rscraper.ListingTopPastWeek)

    if err == nil {

        for _, post := range posts {

            fmt.Println(post.Title)
        }
    }

    . . .

### Retrieve additional pages using the "*after*" Post ID

    . . .

    posts, after, err = rscraper.GetPosts("nfl", rscraper.ListingTypeTop, after, rscraper.ListingTopPastWeek)

    if after == "" {
        // no more pages to load
    }

### Use built-in library constants for easy subreddit listing references

    // Get new posts in a subreddit
    new := rscraper.ListingTypeNew

    // Get top posts over a specified time
    top := rscraper.ListingTypeTop

    // Get hot posts
    hot := rscraper.ListingTypeHot

### Control the time-range for top posts in a subreddit listing

    // Top posts of all time
    allTime := rscraper.ListingTopAllTime

    // Top posts of the past year
    year := rscraper.ListingTopPastYear 

    // Top posts of the past month
    month := rscraper.ListingTopPastMonth

    // Top posts of the past week
    week := rscraper.ListingTopPastWeek 

    // Top posts of the past day
    day := rscraper.ListingTopPastDay 

    // Top posts of the past hour
    hour := rscraper.ListingTopPastHour 

### Get comments from a post 

    comments, after, err := rscraper.GetComments("todayilearned", post.ID, "")

    if err == nil {

        for _, comment := range comments {

            fmt.Println(comment.Body)
        }
    }

    . . . 

### Like posts, use the "*after*" Comment ID to retrieve more comments in a post

    . . .

    comments, after, err = rscraper.GetComments("todayilearned", post.ID, after)

    if after == "" {

        // No more comments
    }

### Comments also have "*after*" IDs stored in them for comment trees that are too deep to traverse at once

    . . .

    for _, reply := range comment.RepliesAfter {

        mComments, mAfter, err := rscraper.GetComments("todayilearned", post.ID, reply)
    }