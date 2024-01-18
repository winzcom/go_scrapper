package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type Node struct {
	next        *Node
	prev        *Node
	value       string
	conditional bool
	command     string
	replacement string
}

type NodeLink struct {
	head *Node
}

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

/**
this is for Nodes
**/

func newLink() *NodeLink {
	return &NodeLink{}
}

func getReplacer(text string) (string, string) {
	reg := regexp.MustCompile(`{{(\.*.+?)\s*}}`)

	all_match := reg.FindAllStringSubmatch(text, -1)
	if len(all_match) > 0 {
		first := all_match[0]
		if len(first) > 0 {
			if len(first) == 2 {
				splitter := strings.Split(first[1], " ")
				if len(splitter) > 1 {
					return splitter[1][1:], splitter[0]
				} else {

					if first[1][0] == '.' {
						return first[1][1:], ""
					} else {
						return "", first[1]
					}
				}
			}
		}
	}
	return ".", ""
}

func insert(text string, n *NodeLink) {
	nln := &Node{
		value: text,
	}
	strip_text := strings.TrimSpace(text)
	if ok, _ := regexp.MatchString(`^{{.*?}}$`, strip_text); ok {
		replacer, command := getReplacer(text)
		if command != "" && replacer != "" {
			nln.conditional = true
			nln.command = command
		} else if command != "" {
			nln.command = command
		}
		nln.replacement = replacer
		Cache = append(Cache, nln)
	}
	if n.head == nil {
		n.head = nln
	} else {
		if n.head.next == nil {
			n.head.next = nln
		} else {
			nh := n.head.next
			for nn := nh; nn != nil; nn = nh.next {
				nh = nn
			}
			nh.next = nln
		}
	}
}

func BuildTextNodes(content string) *NodeLink {
	nl := newLink()

	var str_array []string

	var space_word []byte

	var previous byte

	var ignore_space bool

	//str_array := strings.Split(content, " ")

	for i := 0; i < len(content); i += 1 {
		if content[i] == '{' {
			if previous == content[i] {
				// ignore space
				ignore_space = true
			}
			previous = content[i]
		} else if content[i] == '}' {
			if previous == content[i] {
				ignore_space = false
			}
		}
		if content[i] == ' ' && !ignore_space {
			str_array = append(str_array, string(space_word))
			space_word = []byte{}
		} else {
			space_word = append(space_word, content[i])
		}

		previous = content[i]
	}

	if len(space_word) > 0 {
		str_array = append(str_array, string(space_word))
	}

	for _, v := range str_array {
		insert(v, nl)
	}
	return nl
}

func findNextEnd(l *Node) *Node {
	for {
		if l == nil {
			return nil
		}
		if l.command == "" || l.command != "end" {
			l = l.next
		} else {
			return l.next
		}
	}
}

func findNextElse(tag *tags) *tags {
	if tag != nil {
		if tag.links != nil && tag.links.head.command == "else" {
			return tag
		}
		return findNextElse(tag.next)
	}
	return tag
}

func addNodes(tag *tags, stack []*tags) []*tags {
	v := tag.next
	var ignore bool
	for v != nil {
		if v.links != nil {
			if v.links.head.command == "else" {
				ignore = true
			} else if v.links.head.command == "end" {
				ignore = false
				v = v.next
				continue
			}
		}
		if !ignore {
			stack = append(stack, v)
		}
		v = v.next
	}
	return stack
}

func lookForPrev(tag *tags) []*tags {
	prev := make([]*tags, 0)
	var head *tags = tag
	var counter int
	for head.prev != nil {
		counter += 1
		head = head.prev
	}
	for head != nil && counter > 0 {
		prev = append(prev, head)
		counter -= 1
		head = head.next
	}
	return prev
}

func joinLinks(first, second *NodeLink) *NodeLink {
	if first != nil && second != nil {
		fhead := first.head
		if fhead == nil {
			first = second
		} else {
			shead := second.head
			for fhead.next != nil {

				fhead = fhead.next
			}
			current := fhead
			for shead != nil {
				current.next = shead
				current = current.next
				shead = shead.next
			}
		}
	}
	return first
}

func goLinks(tag *tags, mapper map[string]string, head *Node) (*NodeLink, []*tags) {
	var nl *NodeLink = newLink()
	stack := make([]*tags, 0)
	var ignore bool

	var v *Node
	if head != nil {
		v = head
	} else {
		v = tag.links.head
	}

	for v != nil {
		if v.replacement != "" {
			if v.conditional {
				// look back
				prevs := lookForPrev(tag)
				if mapper[v.replacement] != "" {
					v = v.next
					if v == nil {
						stack = addNodes(tag, stack)
						stack = append(prevs, stack...)
						tag = tag.next
						return nil, stack
					} else {
						fmt.Println("shouid be done ", v.replacement)
						ilinks, _ := goLinks(tag, mapper, v)
						nl = joinLinks(nl, ilinks)
						//insert(v.value, nl)

						v = findNextEnd(v)
						return nl, stack
					}
				} else {
					elsse := findNextElse(tag)
					//fmt.Println("fmt pring ", elsse.links.head, "  opopn ", v)
					if elsse != nil {
						v = elsse.links.head.next
						//fmt.Println("fmt pring ", elsse.links.head, "  opopn ", v)
						if v == nil {
							stack = addNodes(elsse, stack)
							stack = append(prevs, stack...)
							//fmt.Println("houasde ", stack)
							tag = tag.next
							return nil, stack
						} else {
							insert(v.value, nl)
							v = findNextEnd(v)
							return nl, nil
						}
					}
					v = v.next
					for v != nil {
						c := strings.TrimSpace(v.command)
						if c == "" {
							v = v.next
						} else {
							if c == "end" || !v.conditional {
								v = v.next
							}
							break
						}
					}
				}
			} else {
				if mapper[v.replacement] != "" {
					v.value = mapper[v.replacement]
					insert(mapper[v.replacement], nl)
					v = v.next
				} else if v.command != "" {
					insert(v.next.value, nl)
					v = v.next.next
				}
			}
		} else {
			if v.command == "" && !ignore {
				insert(v.value, nl)
			} else if v.command == "else" {
				ignore = true
			} else if v.command == "end" {
				ignore = false
			}
			v = v.next
		}
	}
	return nl, stack
}

func Replacer(root *tags, data interface{}) *tags {
	kindof := reflect.ValueOf(data)
	var mapper map[string]string
	if kindof.Kind() == reflect.Struct {
		total_fields := kindof.NumField()
		for i := 0; i < total_fields; i += 1 {
			name := kindof.Type().Field(i).Name
			val := kindof.Field(i).String()
			if mapper == nil {
				mapper = map[string]string{
					name: val,
				}
			} else {
				mapper[name] = val
			}
		}
	}
	queue := []*tags{root}
	for len(queue) > 0 {
		var tag *tags
		if len(queue) > 0 {
			tag = queue[0]
			if tag.links != nil {
				links, stacks := goLinks(tag, mapper, nil)
				if links != nil {
					tag.links = links
				} else {
					// don't forget previous children

					tag.parent.children = stacks
				}
			}
			queue = queue[1:]
			if tag != nil && len(tag.children) > 0 {
				queue = append(queue, tag.children...)
			}
		}
	}
	return root
}

func Reconstruct(tag *tags) {
	if tag.links != nil {
		head := tag.links.head
		var str_contruct string
		for head != nil {
			str_contruct += head.value + " "
			head = head.next
		}
		tag.content = str_contruct
	}
	for _, v := range tag.children {
		Reconstruct(v)
	}
}

func Rebuild(root *tags, data interface{}) *tags {
	root = Replacer(root, data)
	Reconstruct(root)
	return root
}
