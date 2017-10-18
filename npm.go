package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type npmpackage struct {
	Version string      `json:"version"`
	License interface{} `json:"license"`
	Author  interface{} `json:"author"`
	path    string
}

func (n npmpackage) getLicenseType() string {
	if _, ok := n.License.(string); ok {
		return n.License.(string)
	}
	if _, ok := n.License.(map[string]string); ok {
		m := n.License.(map[string]string)
		return m["type"]
	}
	return ""
}

func (n npmpackage) getAuthor() string {
	if author, ok := n.Author.(string); ok {
		return author
	}
	if m, ok := n.Author.(map[string]interface{}); ok {
		name := m["name"].(string)
		if m["email"] != nil {
			name += "<" + m["email"].(string) + ">"
		}
		return name
	}
	return ""
}

func getNPMPackage(p string) npmpackage {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		log.Fatal(err)
	}
	var pkg npmpackage
	err = json.Unmarshal(b, &pkg)
	if err != nil {
		log.Fatal(err, p)
	}
	pkg.path = p
	return pkg
}
