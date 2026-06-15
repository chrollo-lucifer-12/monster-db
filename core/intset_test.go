package core

import (
	"log"
	"testing"
)

func TestIntset(t *testing.T) {

	is := NewIntset()

	for i := 1; i <= 1000; i++ {
		is.set(int16(i))
	}

	for i := 1; i <= 1000; i++ {
		log.Println(is.search(int16(i)))
	}

}
