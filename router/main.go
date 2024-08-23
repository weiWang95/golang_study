package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	t := NewTree()
	fmt.Println(t.AddRouter("/abc", func() {
		fmt.Println("match -> /abc")
	}))
	fmt.Println(t.AddRouter("/adc", func() {
		fmt.Println("match -> /adc")
	}))
	fmt.Println(t.AddRouter("/adc", func() {
	}))

	for _, path := range []string{
		"/ab",
		"/abc",
		"/abcd",
		"/ad",
		"/adc",
		"/adcb",
	} {
		b, fn := t.Match(path)
		fmt.Printf("%s --> %v\n", path, b)
		if b && fn != nil {
			fn()
		}
	}
}
