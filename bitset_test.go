package bitset

import (
	"math/bits"
	"testing"
)

func TestCountIntersection(test *testing.T) {
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
		test.Error("Bad intersection counts:", setA.CountIntersection(setB), setB.CountIntersection(setA), "should be", count)
	}
	next := setB.CountIntersectionTo(setA, int(count+10))
	if next != count {
		test.Error("Bad first intersection to +10 counts (to", count+10, "):", next, "should be", count)
	}
	next = setA.CountIntersectionTo(setB, int(count+10))
	if next != count {
		test.Error("Bad second intersection to +10 counts (to", count+10, "):", next, "should be", count)
	}
}

func TestPromoteToBitSet(test *testing.T) {
	set := NewIntSet()
	set.Add(2)
	set.Add(3)
	set.Add(4)
	set.Add(5)
	set.Add(6)
	if set.vs != nil {
		test.Error("Promoted while an interval")
	}
	set.promoteToBitSet()
	if set.vs == nil {
		test.Error("Bad promote to bit set:", set.vs, "should not be nil")
	}
	if set.vsStart != 0 {
		test.Error("Bad promote to bit set:", set.vsStart, "should be 0")
	}
	if set.minValue != 2 {
		test.Error("Bad promote to bit set:", set.minValue, "should be 2")
	}
	if set.maxValue != 6 {
		test.Error("Bad promote to bit set:", set.maxValue, "should be 6")
	}
	if set.count != 5 {
		test.Error("Bad promote to bit set:", set.count, "should be 5")
	}
	if bits.OnesCount64(set.vs[0]) != 5 {
		test.Error("Bad promote to bit set:", bits.OnesCount64(set.vs[0]), "should be 5")
	}
}

func TestCountIntersectionIntervals(test *testing.T) {
	setA := NewIntSetFromInterval(1001, 3000)
	setB := NewIntSetFromInterval(101, 2013)
	// intersection is from 1001 to 2013 = 1013
	count := setA.CountIntersection(setB)
	if count != 1013 {
		test.Error("Bad intersection count:", count, "should be 2013")
	}
}

func TestCountIntersectionBitSetInterval(test *testing.T) {
	var count uint
	setA := NewIntSetFromInterval(1001, 3000)
	setB := NewIntSet()
	for j := 101; j < 2013; j += 3 {
		setB.Add(uint(j))
		if setA.Contains(uint(j)) {
			count++
		}
	}
	found := setA.CountIntersection(setB)
	if found != count {
		test.Error("Bad intersection count:", found, "should be", count)
	}

	found = setB.CountIntersection(setA)
	if found != count {
		test.Error("Bad intersection count:", found, "should be", count)
	}
}

func TestIntersect(test *testing.T) {
	setA := NewIntSet()
	setB := NewIntSet()
	intersection := make([]int, 0, 1000)
	for i := 1001; i < 3000; i += 5 {
		setA.Add(uint(i))
	}
	for j := 101; j < 2013; j += 3 {
		setB.Add(uint(j))
		if setA.Contains(uint(j)) {
			intersection = append(intersection, j)
		}
	}
	setA.Intersect(setB)
	for _, j := range intersection {
		if !setA.Contains(uint(j)) {
			test.Error("Bad intersection:", setA.String(), "should contain", j)
			break
		}
	}
	if setA.CountMembers() != uint(len(intersection)) {
		test.Error("Bad intersection count:", setA.CountMembers(), "should be", len(intersection))
	}
}

func TestRemoveAll(test *testing.T) {
	setA := NewIntSet()
	setB := NewIntSet()
	count := 0
	for i := 1001; i < 3000; i += 5 {
		setA.Add(uint(i))
		count++
	}
	for j := 101; j < 2013; j += 3 {
		setB.Add(uint(j))
		if setA.Contains(uint(j)) {
			count--
		}
	}
	setA.RemoveAll(setB)
	if setA.CountIntersection(setB) != 0 {
		test.Error("Bad intersection count:", setA.CountIntersection(setB), "should be 0")
	}
	if setA.CountMembers() != uint(count) {
		test.Error("Bad remaining count:", setA.CountMembers(), "should be", count)
	}
}

func TestMultipleAdd(test *testing.T) {
	set := NewIntSet()
	for i := 0; i < 1000; i++ {
		set.Add(uint(i))
		set.Add(uint(i))
	}
	if set.CountMembers() != 1000 {
		test.Error("Bad count:", set.CountMembers(), "should be 1000")
	}
}

func TestRemove(test *testing.T) {
	set := NewIntSet()
	for i := 0; i < 1000; i++ {
		set.Add(uint(i))
	}
	for i := 250; i < 750; i++ {
		set.Remove(uint(i))
	}
	if set.CountMembers() != 500 {
		test.Error("Bad count:", set.CountMembers(), "should be 500")
	}
}

func TestMultipleRemove(test *testing.T) {
	set := NewIntSet()
	for i := 0; i < 1000; i++ {
		set.Add(uint(i))
	}
	for i := 0; i < 500; i++ {
		set.Remove(uint(i))
		set.Remove(uint(i))
	}
	if set.CountMembers() != 500 {
		test.Error("Bad count:", set.CountMembers(), "should be 500")
	}
}

func TestEmpty(test *testing.T) {
	set := NewIntSet()
	if !set.IsEmpty() {
		test.Error("Bad empty:", set.IsEmpty(), "should be true")
	}
	set.Add(100)
	if set.IsEmpty() {
		test.Error("Bad empty after Add:", set.IsEmpty(), "should be false")
	}
	set.Clear()
	if !set.IsEmpty() {
		test.Error("Bad empty after Clear:", set.IsEmpty(), "should be true")
	}
	set.Add(200)
	if set.IsEmpty() {
		test.Error("Bad empty after Add:", set.IsEmpty(), "should be false")
	}
	set.Remove(200)
	if !set.IsEmpty() {
		test.Error("Bad empty after Remove:", set.IsEmpty(), "should be true")
	}
}

func TestIter(test *testing.T) {
	set := NewIntSet()
	members := make([]int, 0, 1000)
	for i := 200; i < 1000; i += 3 {
		set.Add(uint(i))
		members = append(members, i)
	}

	i := 0
	for ok, id := set.GetFirstID(); ok; ok, id = set.GetNextID(id) {
		if id != uint(members[i]) {
			test.Error("Bad ID:", id, "should be", members[i])
		}
		i++
	}
	if i != len(members) {
		test.Error("Bad count:", i, "should be", len(members))
	}
}

func TestAddAll(test *testing.T) {
	setA := NewIntSet()
	setB := NewIntSet()
	members := make([]int, 0, 1000)
	for i := 1001; i < 3000; i += 5 {
		setA.Add(uint(i))
		members = append(members, i)
	}
	for j := 101; j < 2013; j += 3 {
		setB.Add(uint(j))
		if !setA.Contains(uint(j)) {
			members = append(members, j)
		}
	}
	setA.AddAll(setB)
	for _, m := range members {
		if !setA.Contains(uint(m)) {
			test.Error("Bad members:", setA.String(), "should contain", m)
			break
		}
	}
	if setA.CountMembers() != uint(len(members)) {
		test.Error("Bad count:", setA.CountMembers(), "should be", len(members))
	}
}
