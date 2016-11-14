package main

import (
	"strings"
)

func main() {
	str := "foo:bar::baz"
	for _, p := range strings.Split(str, ":") {
		println("."+p+".", len(p))
	}
}
