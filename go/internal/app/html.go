package app

import (
	"html"
	"regexp"
	"strings"
)

var (
	reBr     = regexp.MustCompile(`<br\s*/?>`)
	rePClose = regexp.MustCompile(`</p>\s*`)
	reH      = regexp.MustCompile(`<h([1-6])>(.*?)</h[1-6]>`)
	reLi     = regexp.MustCompile(`<li>(.*?)</li>`)
	reTag    = regexp.MustCompile(`<[^>]+>`)
)

func htmlToMarkdown(h string) string {
	s := html.UnescapeString(h)
	s = reBr.ReplaceAllString(s, "\n")
	s = rePClose.ReplaceAllString(s, "\n\n")
	s = reH.ReplaceAllString(s, "\n# $2\n")
	s = reLi.ReplaceAllString(s, "- $1\n")
	s = reTag.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
