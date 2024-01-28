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

var Cache []*Node = []*Node{}

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
	"rb":       true,
	"rt":       true,
	"input":    true,
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

var line_counter int

type Attr map[string]string

type AttrLinker map[string]*NodeLink

type tags struct {
	name         string
	children     []*tags
	parent       *tags
	element_type string
	content      string
	text         string
	attributes   Attr
	attr_linker  AttrLinker
	next         *tags
	prev         *tags
	links        *NodeLink
}

var reader *bufio.Reader
var stack []*tags

func noTagAllowed(tag string) bool {
	var slice []string = []string{"<", ">"}

	if slices.Contains(slice, tag) {
		return false
	}
	return true
}

func isNewLineOrReturn(str string) bool {
	if strings.TrimSpace(str) == "" || str == "\r" {
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

	self_line := line_counter

	var is_quoted bool
	//var closing_contents string
	if parent != nil && NOCHILD_ALLOWED[strings.TrimSpace(parent.name)] {
		drop = true
	}

	var last_quote string

	for {
		b, err := reader.ReadByte()
		bts := string(b)
		if bts == "\n" {
			line_counter += 1
			self_line += 1
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Fatal(err)
			}
			break
		}
		if drop {
			p, _ := reader.Peek(2)
			p1, _ := reader.Peek(1)

			if bts == "<" && !is_quoted { //// quotes removed
				if string(p1) == "/" {
					closing = true
					tag_name = ""
					tag.element_type = CLOSING_TAG
				}
			} else if bts == ">" && closing && !is_quoted {
				return tag, true
			} else if (bts == "\"" || bts == "`") && previous != "\\" {
				/***
				Plan is working
				**/

				if is_quoted && last_quote == bts {
					is_quoted = false
					last_quote = ""
				} else if last_quote == "" {
					is_quoted = true
					last_quote = bts
				}
			} else if string(p) == "</" && !is_quoted {
				if letter_count > 0 {
					tag.content = tag_name
					tag.element_type = TEXT
					// build the list of text nodes
					//reader.UnreadByte() //// cancel unread
					return tag, false
				}
				tag.element_type = CLOSING_TAG
				//tag.name = tag_name
				return tag, true
			}
			tag_name += bts
			letter_count += 1
			previous = bts
			continue
		}
		if bts == "<" && !is_comment {
			if !is_tag_opened {
				if !isNewLineOrReturn(tag_name) && letter_count >= 1 { //// increased check to >=

					reader.UnreadByte()
					tag.content = tag_name
					tag.element_type = TEXT
					tag.links = BuildTextNodes(tag_name)
					return tag, false
				}
				if letter_count == 1 {
					fmt.Println("offer a drink ", letter_count, previous)
				}
				is_tag_opened = true
			}
		} else if bts == ">" && !is_quoted { //// another one
			if is_tag_opened || is_comment {
				if is_comment {
					if previous != "-" {
						tag_name += bts
						continue
					}
					tag.element_type = COMMENT
					tag.name = strings.ToLower(COMMENT)
					tag.text = tag_name
					return tag, false
				}
				if closing || (previous == "/") {
					if previous == "/" && !is_quoted {
						tag.element_type = TAG
						if strings.TrimSpace(tag.name) == "" {
							tag.name = strings.TrimSpace(tag_name)
						} else {
							tag.attributes[last_recorded_attr] = strings.TrimRight(tag_name, "/")
							if tag.attr_linker == nil {
								tag.attr_linker = map[string]*NodeLink{
									last_recorded_attr: BuildTextNodes(strings.TrimRight(tag_name, "/")),
								}
							} else {
								tag.attr_linker[last_recorded_attr] = BuildTextNodes(strings.TrimRight(tag_name, "/"))
							}
							//tag.attr_linker[last_recorded_attr] = BuildTextNodes(strings.TrimRight(tag_name, "/"))
						}
						return tag, false /////////////
					}
					if tag_name != "" && !isNewLineOrReturn(tag_name) {
						tag.content = tag_name
					}
					tag.element_type = CLOSING_TAG
					if tag.name == "" && tag_name != "" {
						tag.name = strings.TrimSpace(tag_name)
					}
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
					tag.name = strings.TrimSpace(tag_name)
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
						if tag.attr_linker == nil {
							tag.attr_linker = map[string]*NodeLink{
								last_recorded_attr: BuildTextNodes(strings.TrimRight(tag_name, "/")),
							}
						} else {
							tag.attr_linker[last_recorded_attr] = BuildTextNodes(strings.TrimRight(tag_name, "/"))
						}
					} else if tag_name != "" {
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
				stack = append(stack, tag)
				new_child, _ := readDoc(tag)

				if new_child.name == "" && new_child.content == "" && new_child.element_type != CLOSING_TAG {
					log.Fatalf("Invalid Document seems %v+ does not have closing tag and tag %v", stack[len(stack)-1], tag)
					//return tag, false
				}

				for {
					for new_child != nil && new_child.element_type != CLOSING_TAG {
						//fmt.Println("for every child\n\n", new_child)
						if new_child.name == "" && new_child.content == "" {
							return tag, false
							//log.Fatalf("Invalid Document seems %v+ does not have closing tag", stack[len(stack)-1])
						}
						child_len := len(tag.children)

						if new_child.element_type == TEXT {
							stripped := strings.TrimSpace(new_child.content)
							if len(stripped) > 0 {
								if child_len > 0 {
									last_child := tag.children[child_len-1]
									last_child.next = new_child
									new_child.prev = last_child
								}
								tag.children = append(tag.children, new_child)
							}
						} else {
							if child_len > 0 {
								last_child := tag.children[child_len-1]
								last_child.next = new_child
								new_child.prev = last_child
							}
							tag.children = append(tag.children, new_child)
						}

						new_child, _ = readDoc(tag)
					}
					child_name := strings.TrimSpace(new_child.name)
					if new_child.element_type == CLOSING_TAG && strings.TrimSpace(new_child.name) != "" && strings.TrimSpace(tag.name) != child_name {
						// var last_child *tags
						// var content string
						// if len(tag.children) > 0 {
						// 	last_child = tag.children[len(tag.children)-1]
						// 	if last_child.element_type == TAG {
						// 		content = fmt.Sprintf(
						// 			"after the tag %s with attributes %v+ ", last_child.element_type,
						// 			last_child.attributes,
						// 		)
						// 	} else if last_child.element_type == COMMENT {
						// 		content = fmt.Sprintf("after the comment %s", last_child.text)
						// 	}
						// }
						// log.Fatalf(
						// 	"%q around line %d, has no appropriate closing tag or closed tag %q on line %d, has no opening tag %s",
						// 	stack[len(stack)-1].name,
						// 	self_line+1,
						// 	new_child.name,
						// 	line_counter+1,
						// 	content,
						// )
						new_child, _ = readDoc(tag)
					} else {
						stack = stack[0 : len(stack)-1]
						return tag, false
					}
				}
			}
		} else if bts == "/" && previous == "<" && !is_comment { ////////
			closing = true
			tag_name = ""
			continue //// should probably not continue
		} else if closing {
			previous = bts
			tag_name += bts
			continue
		} else if is_tag_opened {
			//state := is_quoted
			if bts == "\"" {
				if !is_quoted {
					is_quoted = true
				} else {
					is_quoted = false
				}
			}
			if !is_quoted && (bts == " ") { /// where changed was made
				if tag.name == "" {
					tag.name = strings.TrimSpace(tag_name)
				} else if last_recorded_attr != "" {
					if tag.attr_linker == nil {
						tag.attr_linker = map[string]*NodeLink{
							last_recorded_attr: BuildTextNodes(strings.TrimRight(tag_name, "/")),
						}
					} else {
						tag.attr_linker[last_recorded_attr] = BuildTextNodes(strings.TrimRight(tag_name, "/"))
					}
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
	for tag.element_type == COMMENT {
		tag, _ = readDoc(nil)
	}
	return tag
}

func main() {

	// resp, err := http.Get("https://developer.atlassian.com/cloud/jira/platform/getting-started-with-forge/")

	// if err != nil {
	// 	log.Fatal("error gettting url ", err)
	// }

	// defer resp.Body.Close()

	b, _ := os.Open("./html/text.html")

	reader = bufio.NewReader(b)
	root := rootPoint()

	anchors := FindByTag(root, "a")

	fmt.Println("anchild len ", len(anchors))

	for _, v := range anchors {
		fmt.Println("links ", v.attributes["href"])
	}
	//fmt.Println("root ", root.children[0])
	data := struct {
		Runner []int
		Set    string
		Like   string
		Name   string
		Seth   string
		WE     string
	}{
		Runner: []int{1, 2},
		Set:    "like",
		Like:   "set",
		Seth:   "jkl",
		//Name: "FIFA 23",
		WE: "We",
	}
	root = Rebuild(root, data)
	fmt.Println("recontruct ", root.children[0].children[0])
}
