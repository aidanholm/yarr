package feed

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
)

var UnknownFormat = errors.New("unknown feed format")

type processor func(r io.Reader) (*Feed, error)

func sniff(lookup string) (string, processor) {
	lookup = strings.TrimSpace(lookup)
	switch lookup[0] {
	case '<':
		decoder := xml.NewDecoder(strings.NewReader(lookup))
		for {
			token, _ := decoder.Token()
			if token == nil {
				break
			}
			if el, ok := token.(xml.StartElement); ok {
				switch el.Name.Local {
				case "rss":
					return "rss", ParseRSS
				case "RDF":
					return "rdf", ParseRDF
				case "feed":
					return "atom", ParseAtom
				}
			}
		}
	case '{':
		return "json", ParseJSON
	}
	return "", nil
}

func Parse(r io.Reader) (*Feed, error) {
	chunk := make([]byte, 1024)
	if _, err := r.Read(chunk); err != nil {
		return nil, fmt.Errorf("Failed to read input: %s", err)
	}

	_, callback := sniff(string(chunk))
	if callback == nil {
		return nil, UnknownFormat
	}

	r = io.MultiReader(bytes.NewReader(chunk), r)
	return callback(r)
}

func FixURLs(feed *Feed, base string) error {
	baseUrl, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("failed to parse base url: %#v", base)
	}
	siteUrl, err := url.Parse(feed.SiteURL)
	if err != nil {
		return fmt.Errorf("failed to parse feed url: %#v", feed.SiteURL)
	}
	feed.SiteURL = baseUrl.ResolveReference(siteUrl).String()
	for _, item := range feed.Items {
		itemUrl, err := url.Parse(item.URL)
		if err != nil {
			return fmt.Errorf("failed to parse item url: %#v", item.URL)
		}
		item.URL = siteUrl.ResolveReference(itemUrl).String()
	}
	return nil
}