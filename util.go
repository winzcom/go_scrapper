package main

import (
	"fmt"
	"strings"
)

func ItComment(tag_name, bts, previous string) bool {
	/* <![CDATA[ */
	if strings.Contains(tag_name, "!") && bts == "-" && previous == "-" {
		return true
	}
	if bts == "*" && previous == "/" {
		fmt.Println("got my sef a comment")
		return true
	}
	if bts == "/" && previous == "/" {
		return true
	}
	return false
}

func CommentEnds(bts, previous string) bool {
	if bts == "/" && previous == "*" {
		return true
	}
	return false
}
