package area51bot

import (
	"encoding/json"
	"fmt"
	"net/http"

	"os"

	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

const (
	TopicCreatedEventType = "topic_created"
	PostCreatedEventType  = "post_created"
	PostEditedEventType   = "post_edited"
	EventHeader           = "X-Discourse-Event"
	InstanceHeader        = "X-Discourse-Instance"
)

type Notify func(chat int64)

// Post represents Discourse post entry from web hook payload
type Post struct {
	ID              int    `json:"id"`
	AuthorName      string `json:"name"`
	UserName        string `json:"username"`
	Number          int    `json:"post_number"`
	Type            int    `json:"post_type"`
	Preview         string `json:"cooked"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	TopicID         int    `json:"topic_id"`
	TopicSlug       string `json:"topic_slug"`
	DisplayUsername string `json:"display_username"`
	Admin           bool   `json:"admin"`
	Staff           bool   `json:"staff"`
	UserID          int    `json:"user_id"`
}

type Topic struct {
	ID        int          `json:"id"`
	Title     string       `json:"title"`
	CreatedAt string       `json:"created_at"`
	UpdatedAt string       `json:"updated_at"`
	Visible   bool         `json:"visible"`
	UserID    int          `json:"user_id"`
	Slug      string       `json:"slug"`
	Details   *PostDetails `json:"details"`
}

type PostDetails struct {
	CreatedBy *UserDetils `json:"created_by"`
}

type UserDetils struct {
	ID             int    `json:"id"`
	UserName       string `json:"username"`
	AvatarTemplate string `json:"avatar_template"`
}

type User struct {
	ID       int    `json:"id"`
	UserName string `json:"username"`
	Name     string `json:"name"`
}

type DiscoursePayload struct {
	Topic    *Topic `json:"topic"`
	Post     *Post  `json:"post"`
	User     *User  `json:"user"`
	ForumURL string
}

func (p *DiscoursePayload) Message(eventType string) string {
	switch eventType {
	case PostCreatedEventType:
		url := fmt.Sprintf("%s/t/%s/%d/%d", p.ForumURL, p.Post.TopicSlug, p.Post.TopicID, p.Post.ID)
		if p.Topic != nil {
			return fmt.Sprintf("@%s написал новый <a href=\"%s\">пост в \"%s\"</a>", p.Post.UserName, url, p.Topic.Title)
		} else {
			return fmt.Sprintf("@%s написал новый <a href=\"%s\">пост на форум</a>", p.Post.UserName, url)
		}
	case PostEditedEventType:
		url := fmt.Sprintf("%s/t/%s/%d/%d", p.ForumURL, p.Post.TopicSlug, p.Post.TopicID, p.Post.ID)
		if p.Topic != nil {
			return fmt.Sprintf("@%s обновил <a href=\"%s\">пост в \"%s\"</a>", p.Post.UserName, url, p.Topic.Title)
		} else {
			return fmt.Sprintf("@%s обновил <a href=\"%s\">пост на форуме</a>", p.Post.UserName, url)
		}
	case TopicCreatedEventType:
		url := fmt.Sprintf("%s/t/%s/%d", p.ForumURL, p.Topic.Slug, p.Topic.ID)
		return fmt.Sprintf("@%s создал новый топик <a href=\"%s\">\"%s\"</a> на форуме", p.Topic.Details.CreatedBy.UserName, url, p.Topic.Title)
	}

	return ""

}

// HandleDiscourseEvent gets request detects event type and depending of
// that type creates and sends Telegram message
func HandleDiscourseEvent(ctx context.Context, header http.Header, body []byte) (string, error) {
	e := header.Get(EventHeader)
	adr := header.Get(InstanceHeader)

	p := DiscoursePayload{ForumURL: adr}
	err := json.Unmarshal(body, &p)
	if err != nil {
		log.Errorf(ctx, "HandleDiscourseEvent: %s", err.Error())
		return "", err
	}

	// If topic is not in payload get it from the Discourse directly
	if apiKey := os.Getenv("DISCOURSE_API_KEY"); p.Post != nil && p.Topic == nil && apiKey != "" {
		resp, err := urlfetch.Client(ctx).Get(fmt.Sprintf("%s/t/%d.json?api_key=%s&api_username=%s", p.ForumURL, p.Post.TopicID, apiKey, p.Post.UserName))
		if err != nil {
			log.Errorf(ctx, "HandleDiscourseEvent: problem getting Topic details: %s", err.Error())
			return "", err
		}

		t := Topic{}
		err = json.NewDecoder(resp.Body).Decode(&t)
		if err != nil {
			log.Errorf(ctx, "HandleDiscourseEvent: problem unmarshaling Topic body: %s", err.Error())
			return "", err
		}

		p.Topic = &t
	}

	return p.Message(e), nil
}
