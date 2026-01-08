package bitset

import (
	"testing"
)

func Test1CountIntersection(test *testing.T) {
	setA := NewIntSet()
	setB := NewIntSet()
	var count uint
	for i := 1001; i < 3000; i += 5 {
		setA.Add(uint(i))
	}
	for j := 101; j < 2013; j += 3 {
		setB.Add(uint(j))
		if setA.Contains(uint(j)) {
			count++
		}
	}
	if setA.CountIntersection(setB) != count || setB.CountIntersection(setA) != count {
		test.Error("Bad intersection counts:",setA.CountIntersection(setB),setB.CountIntersection(setA),"should be",count)
	}
	next := setB.CountIntersectionTo(setA,int(count+10))
	if next != count {
		test.Error("Bad first intersection to +10 counts (to",count+10,"):",next,"should be",count)
	}
	next = setA.CountIntersectionTo(setB,int(count+10))
	if next != count {
		test.Error("Bad second intersection to +10 counts (to",count+10,"):",next,"should be",count)
	}
}
