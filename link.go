package main

import (
	"log"
	"strings"
)

type Link struct {
	Source      string // Canonicalized link name
	Destination string

	Display string // Entered / display link name (e.g. with dashes)
	Owner   string
}

// maybeFixLinkSource sets the Source of a link to the canonicalized display name of the link.
func maybeFixLinkSource(l *Link) bool {
	canonicalizedDisplay := canonicalizeLink(l.Display)
	if canonicalizedDisplay != l.Source {
		log.Printf("Display link does not canonicalize; forcing canonicalization. old_source='%s' new_source='%s'", l.Source, canonicalizedDisplay)
		l.Source = canonicalizedDisplay
		return true
	}
	return false
}

func canonicalizeLink(l string) string {
	l = strings.ToLower(l)
	l = strings.ReplaceAll(l, ".", "")
	l = strings.ReplaceAll(l, "-", "")
	l = strings.ReplaceAll(l, "_", "")
	return l
}
