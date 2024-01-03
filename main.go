package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strings"
)

const (
	CLOSING_TAG = "CLOSING_TAG"
	TAG         = "TAG"
	TEXT        = "TEXT"
	COMMENT     = "COMMENT"
)

var ALLOWED_TAGS_NO_CLOSE map[string]bool = map[string]bool{
	"br":       true,
	"doctype":  true,
	"meta":     true,
	"img":      true,
	"input":    true,
	"rb":       true,
	"rt":       true,
	"hr":       true,
	"track":    true,
	"iostream": true,
	"source":   true,
	"area":     true,
	"xml":      true,
	"link":     true,
	"base":     true,
	"col":      true,
	"keygen":   true,
	"embed":    true,
	"param":    true,
	"wbr":      true,
}

var NOCHILD_ALLOWED map[string]bool = map[string]bool{
	"script": true,
	"style":  true,
}

type Attr map[string]string

type tags struct {
	name         string
	children     []*tags
	parent       *tags
	element_type string
	content      string
	text         string
	attributes   Attr
}

var reader *bufio.Reader

func noTagAllowed(tag string) bool {
	var slice []string = []string{"<", ">"}

	if slices.Contains(slice, tag) {
		return false
	}
	return true
}

func isNewLineOrReturn(str string) bool {
	if str == " " || str == "\r" {
		return true
	}
	return false
}

func readDoc(parent *tags) (*tags, bool) {
	var tag *tags = &tags{
		element_type: TAG,
		attributes:   Attr{},
		parent:       parent,
	}

	closing := false
	is_tag_opened := false

	var tag_name string
	var previous string
	var last_recorded_attr string
	var letter_count int
	var is_comment bool
	var drop bool
	var possible_comment bool

	var is_quoted bool
	//var closing_contents string
	//var stack []string

	for {
		b, err := reader.ReadByte()
		bts := string(b)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Fatal(err)
			}
			break
		}
		if parent != nil && NOCHILD_ALLOWED[strings.TrimSpace(parent.name)] {
			drop = true
		}
		if drop {
			p, _ := reader.Peek(2)
			p1, _ := reader.Peek(1)

			if bts == "<" && string(p1) == "/" && !is_quoted {
				closing = true
				tag.element_type = CLOSING_TAG
			} else if bts == ">" && closing && !is_quoted {
				return tag, true
			} else if bts == "\"" {
				if is_quoted {
					is_quoted = false
				} else {
					is_quoted = true
				}
			} else if string(p) == "</" {
				if letter_count > 0 {
					tag.content = tag_name
					tag.element_type = TEXT
					reader.UnreadByte()
					return tag, false
				}
				tag.element_type = CLOSING_TAG
				return tag, true
			}
			tag_name += bts
			letter_count += 1
			previous = bts
			continue
		}
		if bts == "<" && !is_comment {
			if !is_tag_opened {
				if !isNewLineOrReturn(tag_name) && letter_count > 1 {
					reader.UnreadByte()
					tag.content = tag_name
					tag.element_type = TEXT
					return tag, false
				}
				is_tag_opened = true
			}
		} else if bts == ">" {
			if is_tag_opened || is_comment {
				if is_comment {
					if previous != "-" {
						continue
					}
					tag.element_type = COMMENT
					tag.name = strings.ToLower(COMMENT)
					tag.text = tag_name
					return tag, false
				}
				if closing || (previous == "/") {
					if previous == "/" {
						tag.element_type = TAG
						if strings.TrimSpace(tag.name) == "" {
							tag.name = tag_name
						} else {
							tag.attributes[last_recorded_attr] = strings.TrimRight(tag_name, "/")

						}
						return tag, false
					}
					if tag_name != "" && !isNewLineOrReturn(tag_name) {
						tag.content = tag_name
					}
					tag.element_type = CLOSING_TAG
					return tag, true
				}
				str_low := strings.ToLower(strings.TrimSpace(tag.name))
				if ALLOWED_TAGS_NO_CLOSE[str_low] {
					tag.element_type = TAG
					if str_low == "doctype" {
						return readDoc(nil)
					}
					return tag, false
				}
				if tag.name == "" {
					tag.name = tag_name
					if tag.name == "" {
						log.Fatal("Invalid Html Document")
					}
					trimmed := strings.TrimSpace(strings.ToLower(tag_name))
					if ALLOWED_TAGS_NO_CLOSE[trimmed] {

						return tag, false
					}
				} else {
					// set up the attributes
					if last_recorded_attr != "" {
						tag.attributes[last_recorded_attr] = tag_name
					} else {
						tag.attributes[tag_name] = ""
					}
					last_recorded_attr = ""
					tag_name = ""
					is_tag_opened = false
					trimmed := strings.TrimSpace(strings.ToLower(tag.name))
					if ALLOWED_TAGS_NO_CLOSE[trimmed] {
						return tag, false
					}
				}
				new_child, _ := readDoc(tag)

				if new_child.name == "" && new_child.content == "" && new_child.element_type != CLOSING_TAG {
					log.Fatal("Invalid Document ", new_child)
				}

				for new_child != nil && new_child.element_type != CLOSING_TAG {

					if new_child.name == "" && new_child.content == "" {
						log.Fatal("Cannot read document ", tag.children[0], new_child)
					}
					if new_child.element_type == TEXT {
						stripped := strings.TrimSpace(new_child.content)
						if len(stripped) > 0 {
							tag.children = append(tag.children, new_child)
						}
					} else {
						tag.children = append(tag.children, new_child)
					}

					new_child, _ = readDoc(tag)
				}

				return tag, false
			}
		} else if bts == "/" && previous == "<" {
			closing = true
		} else if closing {
			previous = bts
			continue
		} else if is_tag_opened {
			if bts == "\"" {
				if !is_quoted {
					is_quoted = true
				} else {
					is_quoted = false
				}
			}
			if bts == " " && !is_quoted {
				if tag.name == "" {
					tag.name = strings.TrimSpace(tag_name)
				} else if last_recorded_attr != "" {
					tag.attributes[last_recorded_attr] = tag_name
				} else {
					tag.attributes[tag_name] = ""
				}
				tag_name = ""
				last_recorded_attr = ""
				continue
			} else if bts == "=" && is_tag_opened && !is_quoted {
				last_recorded_attr = tag_name
				tag_name = ""
				continue
			}
		}
		if bts == "-" && is_comment {
			previous = bts
			continue
		}
		if !is_tag_opened && !isNewLineOrReturn(bts) {
			letter_count += 1
			tag_name += bts
		} else if (bts == "!" || bts == "?") && previous == "<" {
			possible_comment = true
			continue
		} else if noTagAllowed(bts) {
			tag_name += bts
		}
		if possible_comment {
			if bts == "-" && previous == "-" && !is_comment {
				is_comment = true
				is_tag_opened = false
				tag_name = ""
			}
		}
		previous = bts
	}
	return tag, false
}

func rootPoint() *tags {
	tag, _ := readDoc(nil)
	if tag.element_type != COMMENT {
		return tag
	}
	tag, _ = readDoc(nil)
	return tag
}

func main() {
	b, _ := os.Open("./html/sporty.html")

	reader = bufio.NewReader(b)
	root := rootPoint()

	anchors := FindByTag(root, "a")

	fmt.Println("anchild len ", len(anchors))

	// for _, v := range anchors {
	// 	fmt.Println("a ", v.attributes["href"])
	// }
	fmt.Println("root ", root)
	fmt.Println("find by attribute ", FindByKey(root, "id", "army"))
	fmt.Println("find by tags ", len(FindByTag(root, "script")))
}
