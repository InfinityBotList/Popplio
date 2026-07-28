package seo

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type RssFeed struct {
	XMLName xml.Name `xml:"rss"`
	// NS
	NS string `xml:"xmlns:atom,attr"`
	// Version
	Version string `xml:"version,attr"`
	// Channel
	Channel *RssChannel `xml:"channel"`
}

type RssChannel struct {
	Title         string         `xml:"title"`
	Link          string         `xml:"link"`
	Description   string         `xml:"description"`
	Language      string         `xml:"language"`
	LastBuildDate string         `xml:"lastBuildDate"`
	Copyright     string         `xml:"copyright"`
	Docs          string         `xml:"docs"`
	TTL           int            `xml:"ttl"`
	Category      []string       `xml:"category"`
	AtomLink      []*RssAtomLink `xml:"atom:link"`
	Links         []*RssLink
	Generator     string     `xml:"generator"`
	Image         *RssImage  `xml:"image"`
	Items         []*RssItem `xml:"item"`
}

type RSSGuid struct {
	XMLName     xml.Name `xml:"guid"`
	Guid        string   `xml:",chardata"`
	IsPermaLink bool     `xml:"isPermaLink,attr"`
}

type RssAtomLink struct {
	XMLName xml.Name `xml:"atom:link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr"`
	Type    string   `xml:"type,attr"`
}

type RssImage struct {
	XMLName xml.Name `xml:"image"`
	// URL
	URL string `xml:"url"`
	// Title
	Title string `xml:"title"`
	// Link
	Link string `xml:"link"`
}

type RssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Author      string   `xml:"author"`
	Category    string   `xml:"category"`
	Guid        *RSSGuid `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
	Updated     string   `xml:"updated"`
}

type RssLink struct {
	XMLName xml.Name `xml:"link"`
	Rel     string   `xml:"rel,attr"`
	Href    string   `xml:"href,attr"`
}

// Adds a RSS Item to a feed
func (m *MapGenerator) AddToRss(ctx context.Context, f Fetcher, feed *RssFeed, category, id string) error {
	e, err := m.Add(ctx, f, id)

	if err != nil {
		return err
	}

	feed.Channel.Items = append(feed.Channel.Items, &RssItem{
		Title:       e.Name,
		Link:        e.URL,
		Description: e.Description,
		Author: func() string {
			if e.Author != nil {
				return fmt.Sprintf("%s %s [%s]", cases.Title(language.AmericanEnglish).String(e.Author.Type), e.Author.Name, e.Author.ID)
			} else {
				return "Unknown Author"
			}
		}(),
		Category: category,
		Guid: &RSSGuid{
			Guid:        e.ID,
			IsPermaLink: false,
		},
		PubDate: e.CreatedAt.Format(time.RFC822),
		Updated: e.UpdatedAt.Format(time.RFC822),
	})

	return nil
}
