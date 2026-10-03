package core

import (
	"context"
	"strconv"
	"testing"
)



func makeSaddArgs(n int) []string {
	args := make([]string, n+1)
	args[0] = "myset"
	for i := 1; i <= n; i++ {
		args[i] = strconv.Itoa(i)
	}
	return args
}

func resetStore() {
	
	store = make(map[string]Obj) 
}

func BenchmarkSAdd(b *testing.B) {
	ctx := context.Background()
	c := &benchClient{}

	args := makeSaddArgs(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetStore()
		SaddCmd{}.Execute(ctx, c, args)
	}
}

func BenchmarkSRem(b *testing.B) {
	ctx := context.Background()
	c := &benchClient{}

	args := makeSaddArgs(100)

	for i := 0; i < 100; i++ {
		SaddCmd{}.Execute(ctx, c, args)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SremCmd{}.Execute(ctx, c, []string{"myset", "50"})
	}
}


func BenchmarkSIsMember(b *testing.B) {
	ctx := context.Background()
	c := &benchClient{}

	args := makeSaddArgs(100)
	SaddCmd{}.Execute(ctx, c, args)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SismemberCmd{}.Execute(ctx, c, []string{"myset", "50"})
	}
}


func BenchmarkSCard(b *testing.B) {
	ctx := context.Background()
	c := &benchClient{}

	args := makeSaddArgs(100)
	SaddCmd{}.Execute(ctx, c, args)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScardCmd{}.Execute(ctx, c, []string{"myset"})
	}
}


func BenchmarkSMembers(b *testing.B) {
	ctx := context.Background()
	c := &benchClient{}

	args := makeSaddArgs(100)
	SaddCmd{}.Execute(ctx, c, args)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SmembersCmd{}.Execute(ctx, c, []string{"myset"})
	}
}
