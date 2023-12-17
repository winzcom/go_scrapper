package main

import (
	"log"
	"strings"
)

func FindByKey(root *tags, key, value string) []tags {
	var keys []tags

	if root == nil {
		log.Fatal("root is nil")
	}
	stack := []*tags{root}
	ll := len(stack)
	for i := 0; i < ll; i += 1 {
		v := stack[i]
		if val, ok := v.attributes[key]; ok {
			replacement := strings.ReplaceAll(val, "\"", "")
			if replacement == value {
				keys = append(keys, *v)
			}
		}
		if len(v.children) > 0 {
			for _, c := range v.children {
				stack = append(stack, c)
				ll = len(stack)
			}
		}
	}
	return keys
}

func FindByTag(root *tags, tag_name string) []tags {
	var keys []tags
	if root == nil {
		log.Fatal("root is nil")
	}
	stack := []*tags{root}
	ll := len(stack)
	for i := 0; i < ll; i += 1 {
		v := stack[i]
		s_l := strings.ToLower(tag_name)
		if s_l == strings.TrimSpace(v.name) {
			keys = append(keys, *v)
		}
		if len(v.children) > 0 {
			for _, c := range v.children {
				stack = append(stack, c)
				ll += 1
			}
		}
	}
	return keys
}

func LookForText(root *tags, sample string) []tags {
	var keys []tags

	stack := []*tags{root}
	ll := len(stack)
	for i := 0; i < ll; i += 1 {
		v := stack[i]
		if v.content != "" {
			stripped_content := strings.ReplaceAll(v.content, " ", "")
			stripped_sample := strings.ReplaceAll(sample, " ", "")

			if strings.Contains(stripped_content, stripped_sample) {
				keys = append(keys, *v)
			}
		}
		if len(v.children) > 0 {
			for _, c := range v.children {
				stack = append(stack, c)
				ll += 1
			}
		}
	}
	return keys
}
