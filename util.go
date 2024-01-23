package main

import (
	"fmt"
	"log"
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

//var prev []*tags

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

func findNextNodeEnd(tag *tags) *tags {
	cur := tag
	if cur != nil && cur.next != nil {
		//fmt.Println("foind ened ", cur.next)
		if cur.links != nil && (cur.links.head.command == "end") {
			//fmt.Println("foind ened ", cur)
			return cur
		}
		return findNextNodeEnd(cur.next)
	}
	return cur
}

func findNextElse(tag *tags) *tags {
	cur := tag
	if cur.next != nil {
		cur = cur.next
		if cur.links != nil && (cur.links.head.command == "else" || cur.links.head.command == "elif" || cur.links.head.command == "end") {
			return cur
		}
		return findNextElse(cur)
	}
	return cur
}

func addNodes(tag *tags, stack []*tags) []*tags {
	//fmt.Println("who called you ", tag)
	v := tag.next
	var ignore bool
	for v != nil {
		if v.links != nil {
			if v.links.head.command != "" {
				if v.links.head.command == "end" {
					ignore = false
					v = v.next
					continue
				}
				ignore = true
			}
		}
		if !ignore {
			stack = append(stack, v)
		}
		v = v.next
	}
	return stack
}

func addNodesUntilEnd(tag *tags, stack []*tags) ([]*tags, *tags) {
	//fmt.Println("who called you ", tag)
	v := tag.next
	var ignore bool
	for v != nil {
		if v.links != nil {
			if v.links.head.command != "" {
				if v.links.head.command == "end" {
					break
				}
				ignore = true
			}
		}
		if !ignore {
			stack = append(stack, v)
		}
		v = v.next
	}
	return stack, v
}

func lookForPrev(tag *tags) []*tags {
	prev := make([]*tags, 0)

	var head *tags

	head = tag
	var counter int
	for head.prev != nil {
		//fmt.Println("head ", head, counter)
		head = head.prev
		counter += 1
	}
	for head != nil && counter > 0 {
		prev = append(prev, head)
		counter -= 1
		head = head.next
	}
	//fmt.Println("prevv ", prev)
	return prev
}

func reworkLinks(stack []*tags) []*tags {
	ll := len(stack)
	for i, _ := range stack {
		if i+1 < ll {
			stack[i].next = stack[i+1]
			stack[i+1].prev = stack[i]
		}
	}
	return stack
}

func joinLinks(first, second *NodeLink) *NodeLink {
	nl := newLink()
	//nl.head = &Node{}
	if first != nil && second != nil {
		fhead := first.head
		if fhead == nil {
			first = second
		} else {
			shead := second.head
			//fmt.Println("lucky boy ", fhead)
			if nl.head == nil {
				nl.head = first.head
			}
			nl.head.value = first.head.value
			nlhead := nl.head
			for fhead.next != nil {
				fhead = fhead.next
				nlhead.next = fhead
				nlhead = nlhead.next
			}
			current := nl.head
			for shead != nil {
				current.next = shead
				shead = shead.next
			}
		}
	}
	return nl
}

func findAllSiblings(tag tags) []*tags {
	cur := tag.next
	var new_stack []*tags = []*tags{&tag}
	for cur != nil && (cur.links == nil || cur.links.head == nil) {
		new_stack = append(new_stack, cur)
		cur = cur.next
	}
	return new_stack
}

func refix(tag []*tags) {
	var queue []*tags

	if len(tag) > 0 {
		queue = []*tags{tag[0]}
		for len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			if v.links != nil && v.links.head != nil {
				cp := *(v.links.head)
				v.links.head = &cp
			}
			if len(v.children) > 0 {
				queue = append(queue, v.children...)
			}
		}
	}
}

func walkLinks(link *Node) []*Node {
	var replacers_commands []*Node

	not_allowed_command := map[string]bool{
		"elif": true,
		"else": true,
	}

	var ignore bool

	cur := link.next
	for cur != nil {
		if cur.command != "" {
			if not_allowed_command[cur.command] {
				ignore = true
			}
		}
		if !ignore {
			replacers_commands = append(replacers_commands, cur)
		}
		cur = cur.next
	}
	return replacers_commands
}

func refixLinks(nodes []*Node) *NodeLink {
	if len(nodes) == 0 {
		return nil
	}
	nl := newLink()
	var cur *Node
	for _, v := range nodes {
		if nl.head == nil {
			nl.head = v
			cur = nl.head
		} else {
			cur.next = v
			cur = cur.next
		}
	}
	return nl
}

func subcommands(command string, tag *tags, link Node, mapper map[string]interface{}, nl *tags) []*tags {
	if command == "range" {
		repl := link.replacement

		val, ok := mapper[repl]

		if !ok || val == nil {
			return []*tags{}
		}
		ll := len(val.([]int)) - 1

		nls := make([]*tags, 0)
		if nl != nil {
			nls = append(nls, nl)
		}
		stack, end_tag := addNodesUntilEnd(tag, nls)

		new_stack := make([]*tags, 0)

		for ; ll >= 0; ll -= 1 {
			new_stack = append(new_stack, stack...)
		}
		new_stack = append(new_stack, addNodes(end_tag, make([]*tags, 0))...)
		fmt.Println("new linker ", tag.links.head.next)
		return new_stack
	}
	return []*tags{}
}

func goLinks(tag *tags, mapper map[string]interface{}, head *Node, prev []*tags) (*NodeLink, []*tags) {
	replacement := tag.links.head.replacement

	link := tag.links
	if replacement == "" {
		log.Fatal("Provide a condition to check for")
	}
	value, ok := mapper[replacement]

	if ok && value != nil {
		// find the elements within range of
		if len(prev) == 0 {
			// get all previous nodes
			prev = lookForPrev(tag)
		}
		if link.head != nil && link.head.next != nil {
			replacers_command := walkLinks(link.head)
			// if len(commands) == 0 {
			// 	replaceContent(re)
			// }
			// traverse the links
			new_tag_link := newLink()
			if len(replacers_command) > 0 {

				crl := len(replacers_command)
				for i, v := range replacers_command {
					if v.command == "" {
						_, ok := mapper[v.replacement]
						if !ok {
							log.Fatalf("\nno value received for %s ", v.replacement)
						}
						insert(mapper[v.replacement].(string), new_tag_link)
					} else {
						if i+1 < crl {
							repl := v.replacement
							val, ok := mapper[repl]
							if ok && val != nil {
								//lel := len(val.([]int))
								wlinks := walkLinks(v)
								fmt.Println("washt sls ", wlinks)
								nl := refixLinks(wlinks)
								for _, v := range wlinks {
									if v.command == "" && v.replacement != "" {
										_, ok := mapper[v.replacement]
										if !ok {
											log.Fatalf("\nno value received for %s ", v.replacement)
										}
										insert(mapper[v.replacement].(string), nl)
									}
								}
								new_tag_link = joinLinks(new_tag_link, nl)
								tag.links.head = new_tag_link.head
								// are we done
								var stack []*tags
								if tag.next != nil {
									var ts *tags
									if nl != nil {
										ts = tag
									}
									stack = subcommands(v.command, tag, *v, mapper, ts)
								}
								stack = append(prev, stack...)
								return nil, stack
							}
						} else {
							var ts *tags
							if new_tag_link.head != nil {
								ts = tag
								ts.links = new_tag_link
							}
							stack = subcommands(v.command, tag, *v, mapper, ts)
							stack = append(prev, stack...)
						}
						break
					}
				}
				var ts *tags
				if new_tag_link.head != nil {
					ts = tag
					ts.links = new_tag_link
				}
				tag.links = new_tag_link
				new_s := make([]*tags, 0)
				if ts != nil {
					new_s = append(new_s, ts)
				}
				nk, end_t := addNodesUntilEnd(tag, new_s)
				stack = append(stack, nk...)
				stack = append(stack, addNodes(end_t, make([]*tags, 0))...)
				stack = append(prev, stack...)
				return nil, stack
			}
		} else {
			stack = subcommands(link.head.command, tag, *link.head, mapper, nil)
			stack = append(prev, stack...)
		}
	}
	return nil, stack
}

func ttr(tag *tags, mapper map[string]interface{}, prev []*tags) *tags {
	if tag.links != nil {
		var links *NodeLink
		//var stacks []*tags
		var prev []*tags
		links, stack = goLinks(tag, mapper, nil, prev)
		if links != nil {
			tag.links = links
		}
		if len(stack) > 0 {
			tag.parent.children = reworkLinks(stack)
		}
	}
	for _, v := range tag.children {
		ttr(v, mapper, prev)
	}
	return tag
}

func createMapper(data interface{}) map[string]interface{} {
	kindof := reflect.ValueOf(data)
	var mapper map[string]interface{}

	if kindof.Kind() == reflect.Struct {
		total_fields := kindof.NumField()
		for i := 0; i < total_fields; i += 1 {
			//var value_of interface{}
			name := kindof.Type().Field(i).Name
			val := kindof.Field(i)
			if kindof.Type().Kind() == reflect.Slice {
				if !kindof.Type().Field(i).IsExported() {
					log.Fatal("Cannot not have a slice not exported")
				}
				//value_of = val.Interface()
			}
			if mapper == nil {
				mapper = map[string]interface{}{
					name: val.Interface(),
				}
			} else {
				mapper[name] = val.Interface()
			}
		}
	}
	return mapper
}
func Replacer2(root *tags, data interface{}) *tags {
	mapper := createMapper(data)
	prev := make([]*tags, 0)
	return ttr(root, mapper, prev)
}

func Replacer(root *tags, data interface{}) *tags {
	kindof := reflect.ValueOf(data)
	var mapper map[string]interface{}

	if kindof.Kind() == reflect.Struct {
		total_fields := kindof.NumField()
		for i := 0; i < total_fields; i += 1 {
			//var value_of interface{}
			name := kindof.Type().Field(i).Name
			val := kindof.Field(i)
			if kindof.Type().Kind() == reflect.Slice {
				if !kindof.Type().Field(i).IsExported() {
					log.Fatal("Cannot not have a slice not exported")
				}
				//value_of = val.Interface()
			}
			if mapper == nil {
				mapper = map[string]interface{}{
					name: val.Interface(),
				}
			} else {
				mapper[name] = val.Interface()
			}
		}
	}
	// get all linkers
	var linkers []*tags

	linker_map := map[string]bool{
		"else": true,
		"elif": true,
		"end":  true,
	}

	queue := []*tags{root}
	for len(queue) > 0 {
		var tag *tags
		if len(queue) > 0 {
			tag = queue[0]
			if tag.links != nil {
				if tag.links.head.command != "" && linker_map[tag.links.head.command] == false {
					linkers = append(linkers, tag)
				}
			}
			queue = queue[1:]
			if tag != nil && len(tag.children) > 0 {
				queue = append(queue, tag.children...)
			}
		}
	}
	fmt.Println("linkers size ", len(linkers))
	for _, v := range linkers {
		var prev []*tags
		//fmt.Println("linkers ", v)
		links, stack := goLinks(v, mapper, nil, prev)
		if links != nil {
			v.links = links
		} else {
			if len(stack) > 0 {
				v.parent.children = reworkLinks(stack)
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
