package seo

import (
	"context"
	"encoding/xml"
	"fmt"
)

// A standard xml sitemap
/*<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://www.example.com/foo.html</loc>
    <lastmod>2022-06-04</lastmod>
  </url>
</urlset>
*/
type Sitemap struct {
	XMLName xml.Name      `xml:"urlset"`
	XMLNS   string        `xml:"xmlns,attr"`
	Urls    []*SitemapURL `xml:"url"`
}

// URL is a structure of <url> in <sitemap>
type SitemapURL struct {
	XMLName     xml.Name `xml:"url"`
	Name        string   `xml:"name,omitempty"`        // Extra field, not part of the standard
	Category    string   `xml:"category,omitempty"`    // Extra field, not part of the standard
	Description string   `xml:"description,omitempty"` // Extra field, not part of the standard
	Loc         string   `xml:"loc"`
	ChangeFreq  string   `xml:"changefreq"`
	LastMod     string   `xml:"lastmod"`
	Priority    string   `xml:"priority"`
}

// Adds a sitemap item to a feed
func (m *MapGenerator) AddToSitemap(ctx context.Context, f Fetcher, sitemap *Sitemap, category, id string, priority float64) error {
	e, err := m.Add(ctx, f, id)

	if err != nil {
		return err
	}

	sitemap.Urls = append(sitemap.Urls, &SitemapURL{
		Name:        e.Name,
		Category:    category,
		Description: e.Description,
		Loc:         e.URL,
		ChangeFreq:  "daily",
		LastMod:     e.UpdatedAt.Format("2006-01-02"),
		Priority:    fmt.Sprint(priority),
	})

	return nil
}
