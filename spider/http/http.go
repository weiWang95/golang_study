package http

import (
	"bytes"
	"net/http"
	"time"
	"io"
	"os"
)

func getClient() *http.Client {
	return &http.Client{ Timeout: 5 * time.Second }
}

func Get(url string) string {
	client := getClient()

	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var buffer [1024]byte
	result := bytes.NewBuffer(nil)

	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}

	return result.String()
}

func Download(url string, file *os.File) {
	client := getClient()

	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var buffer [1024]byte

	for {
		n, err := resp.Body.Read(buffer[0:])
		file.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}
}

