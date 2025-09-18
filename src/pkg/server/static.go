package server

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var content embed.FS

var staticContent fs.FS

func init() {
	// Create a sub-FS rooted at "static"
	var err error
	staticContent, err = fs.Sub(content, "static")
	if err != nil {
		panic(err)
	}
}
