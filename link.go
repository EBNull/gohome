package main

import (
	"strings"
)

type Link struct {
	Source      string // Canonicalized link name
	Destination string

	Display string // Entered / display link name (e.g. with dashes)
	Owner   string
}

func canonicalizeLink(l string) string {
	l = strings.ToLower(l)
	l = strings.ReplaceAll(l, ".", "")
	l = strings.ReplaceAll(l, "-", "")
	l = strings.ReplaceAll(l, "_", "")
	return l
}
