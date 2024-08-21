package http

import (
	"fmt"
	"errors"
	"os"
	"regexp"
)

type Image struct {
	Uri string
	FullName string
	Version string
	Uid string
	Quality string
	FileType string
}

func (image Image) GetQuality() int {
	switch image.Quality {
		case "r":
			return 200
		case "xs":
			return 120
		case "hd":
			return 100
		case "1440w":
			return 80
		case "720w":
			return 30
		case "b":
			return 10
		case "l":
			return 5
		default:
			return 1
	}
}

func (image Image) Download(basePath string) {
	os.MkdirAll(basePath, os.ModePerm)

	filePath := fmt.Sprintf("%s/%s", basePath, image.FullName)

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Create File Failed -> %+v\n", err)
		return
	}
	defer file.Close()

	Download(image.Uri, file)

	file.Sync()
}

func NewImage(uri string) (*Image, error) {
	re := regexp.MustCompile(`[\.\w\-\:\/]+\/((\w+)\-(\w+)\_(\w+)\.(\w+))`)
	res := re.FindAllStringSubmatch(uri, -1)

	if len(res) != 1 || len(res[0]) != 6 {
		msg := fmt.Sprintf("Create image failed: bad image uri -> %s => %+v\n", uri, res)
		return nil, errors.New(msg)
	}
	data := res[0]

	return &Image{ Uri: uri, FullName: data[1], Version: data[2], Uid: data[3], Quality: data[4], FileType: data[5] }, nil
}
