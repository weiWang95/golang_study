package main

import (
	"fmt"
)

type WebUrl struct {
	urlType string
	url string
}

type Image struct {
	WebUrl
	name string
}

func NewImage(name string, url string) *Image {
	return &Image{name: name, WebUrl: WebUrl{urlType: "Image", url: url}}
}

func (img *Image) download() {
	fmt.Println("Image download ...")
}
