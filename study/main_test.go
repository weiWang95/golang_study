package main

import "testing"

func BenchmarkIsOdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsOdd(i)
	}
}

func BenchmarkIsOdd2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsOdd2(i)
	}
}

func IsOdd(n int) bool {
	return n&1 == 1
}

func IsOdd2(n int) bool {
	return n%2 != 0
}
