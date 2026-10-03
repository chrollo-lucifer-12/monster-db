package core

import (
	"math/rand"
	"time"
)

const DecayMinutes = 2

func LFUClock() uint32 {
	return uint32(time.Now().Unix() / 60)
}

func getCounter(obj Obj) uint8 {
	return uint8(obj.LastAccessedAt)
}

func getLastDecay(obj Obj) uint32 {
	return obj.LastAccessedAt >> 8
}

func setLFU(k string, obj Obj, t uint32, counter uint8) {
	obj.LastAccessedAt = (t << 8) | uint32(counter)
	store[k] = obj
}

func decay(counter uint8, periods uint32) uint8 {
	for periods > 0 && counter > 0 {
		counter /= 2
		periods--
	}

	return counter
}

func increment(counter uint8) uint8 {
	if counter == 255 {
		return 255
	}

	p := 1.0 / float64(counter+1)

	if rand.Float64() < p {
		counter++
	}

	return counter
}

func touch(obj Obj) Obj {
	now := LFUClock()
	counter := getCounter(obj)
	last := getLastDecay(obj)

	elapsed := now - last
	periods := elapsed / DecayMinutes
	counter = decay(counter, periods)
	counter = increment(counter)

	obj.LastAccessedAt = (now << 8) | uint32(counter)
	return obj
}
