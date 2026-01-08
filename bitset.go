package bitset

import (
	"fmt"
	"math/bits"
)

const (
	Bit     uint64 = 1
	AllBits uint64 = 0xFFFFFFFFFFFFFFFF
)

type IntSet struct {
	vs    []uint64
	vsStart uint //value offset, a multiple of 64
	start uint // index in vs
	end   uint
	count uint //size is no greater than this
}

func NewIntSet() *IntSet {
	set := IntSet{vs: make([]uint64, 50), vsStart:0, start: 1, end: 0, count: 0}
	return &set
}

func NewIntSetCapacity(capacity int) *IntSet {
	set := IntSet{vs: make([]uint64, capacity/64+1), vsStart:0, start: 1, end: 0, count: 0}
	return &set
}

func NewIntSetFromInts(values []int) *IntSet {
	var max int
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	set := IntSet{vs: make([]uint64, max/64+1), vsStart: 0, start: 1, end: 0, count: 0}
	s := &set
	s.AddInts(values)
	return s
}

func NewIntSetFromUInts(values []uint) *IntSet {
	var max uint
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	set := IntSet{vs: make([]uint64, max/64+1), vsStart:0, start: max / 64, end: 0, count: 0}
	s := &set
	for _, v := range values {
		s.Add(v)
	}
	return s
}

func (set *IntSet) AddInts(values []int) {
	for _, v := range values {
		set.Add(uint(v))
	}
}

func (set *IntSet) Contains(x uint) bool {
	if x < set.vsStart {
		return false
	}
	index := (x-set.vsStart) >> 6
	if index < set.start || index > set.end {
		return false
	}
	subIndex := x & 0x3F
	return (set.vs[index] & (Bit << subIndex)) != 0
}

func (set *IntSet) Add(x uint) {
	if x < set.vsStart {
		//TODO: re-allocate the vs slice and update vsStart
	}
	index := (x-set.vsStart) >> 6
	subIndex := x & 0x3F
	bit := Bit << subIndex
	if int(index) >= len(set.vs) {
		newVs := make([]uint64, index+2)
		copy(newVs, set.vs)
		set.vs = newVs
	}
	if set.end < set.start {
		set.start = index
		set.end = index
		set.vs[index] = bit
		set.count = 1
		return
	}
	if index < set.start {
		set.start = index
		set.vs[index] = bit
		set.count++
		return
	}
	if index > set.end {
		set.end = index
		set.vs[index] = bit
		set.count++
		return
	}
	old := set.vs[index]
	if (old & bit) != 0 { //already exists
		return
	}
	set.vs[index] = old | bit
	set.count++
}

func (set *IntSet) Remove(x uint) {
	if x < set.vsStart {
		return
	}
	index := (x-set.vsStart) >> 6
	if index > set.end || index < set.start {
		return
	}
	subIndex := x & 0x3F
	bit := Bit << subIndex
	old := set.vs[index]
	if (old & bit) == 0 {
		return //nothing to remove
	}
	set.vs[index] ^= bit
	if index == set.start || index == set.end {
		set.reduceStartEnd()
	}
	set.count--
}

func (set *IntSet) reduceStartEnd() {
	for set.start <= set.end && set.vs[set.start] == 0 {
		set.start++
	}
	for set.end >= set.start && set.vs[set.end] == 0 {
		set.end--
	}
	if set.start > set.end {
		set.start = uint(len(set.vs)) + 1
		set.end = 0
	}
}

func (set *IntSet) IsEmpty() bool {
	return set.start > set.end
}

func (set *IntSet) Clear() {
	for set.start <= set.end {
		set.vs[set.start] = 0
		set.start++
	}
	set.end = 0
	set.start = uint(len(set.vs)) + 1
	set.count = 0
}

func (set *IntSet) GetFirstID() (bool, uint) {
	if set.IsEmpty() {
		return false, 0
	}
	v := set.vs[set.start]
	return true, set.vsStart + set.start*64 + uint(bits.TrailingZeros64(v))
}

func (set *IntSet) CountIntersection(other *IntSet) uint {
	start := set.start
	end := set.end
	if other.start > start {
		start = other.start
	}
	if end > other.end {
		end = other.end
	}
	count := 0
	for ; start <= end; start++ {
		count += bits.OnesCount64(set.vs[start] & other.vs[start])
	}
	return uint(count)
}

func (set *IntSet) CountIntersectionTo(other *IntSet, maxCount int) uint {
	start := set.start
	end := set.end
	if other.start > start {
		start = other.start
	}
	if end > other.end {
		end = other.end
	}

	count := 0
	for ; start <= end && count < maxCount; start++ {
		count += bits.OnesCount64(set.vs[start] & other.vs[start])
	}
	return uint(count)
	//return countIntersectionToAsm(set.vs[start:end+1], other.vs[start:end+1], maxCount)
}

//func countIntersectionToAsm(a, b []uint64, maxCount int) uint

func (set *IntSet) Intersect(other *IntSet) {
	a := set.vsStart >> 6
	b := other.vsStart >> 6
	for ; set.start + b < other.start + a && set.start <= set.end; set.start++ {
		set.vs[set.start] = 0
	}
	for ; set.end + b > other.end + a && set.end >= set.start; set.end-- {
		set.vs[set.end] = 0
	}
	for i := set.start; i <= set.end; i++ {
		set.vs[i] &= other.vs[i+a-b]
	}
	set.reduceStartEnd()
}

func (set *IntSet) RemoveAll(other *IntSet) {
	a := set.vsStart >> 6
	b := other.vsStart >> 6
	start := set.start
	end := set.end
	if other.start + a > start + b {
		start = other.start + a - b
	}
	if end + b > other.end + a {
		end = other.end
	}
	for i := start; i <= end; i++ {
		set.vs[i] &= (^other.vs[i + a - b])
	}
	set.reduceStartEnd()
}

func (set *IntSet) AddAll(other *IntSet) {
	a := set.vsStart >> 6
	b := other.vsStart >> 6
	start := other.start
	end := other.end
	empty := set.start > set.end
	if start + a < set.start + b || empty {
		// extend earlier
		if set.start + a >= b {
			set.start = set.start + a - b
		} else {
			// TODO: reallocate
		}
	}
	//TODO: adjust by a and b below as well
	if end > set.end || empty {
		if end >= uint(len(set.vs)) {
			newVs := make([]uint64, end+1)
			copy(newVs, set.vs)
			set.vs = newVs
		}
		set.end = end
	}
	for i := start; i <= end; i++ {
		set.vs[i] |= other.vs[i]
	}
}

func (set *IntSet) Union(other *IntSet) *IntSet {
	return nil
}

func (set *IntSet) Intersection(other *IntSet) *IntSet {
	return nil
}

func (set *IntSet) GetNextID(x uint) (bool, uint) {
	x++
	index := x >> 6
	if index > set.end {
		return false, 0
	}
	subIndex := uint(0)
	if index < set.start {
		index = set.start
	} else {
		subIndex = x & 0x3F
		filter := AllBits << subIndex
		if (filter & set.vs[index]) == 0 {
			index++
			subIndex = 0
		}
	}
	for index <= set.end && set.vs[index] == 0 {
		index++
	}
	if index > set.end {
		return false, 0
	}
	v := set.vs[index] >> subIndex
	zs := uint(bits.TrailingZeros64(v))
	return true, (index << 6) + subIndex + zs
}

func (set *IntSet) AsInts() []int {
	//consider inlining the loop to speed this up
	ids := make([]int, 0, set.count)
	for ok, id := set.GetFirstID(); ok; ok, id = set.GetNextID(id) {
		ids = append(ids, int(id))
	}
	return ids
}
func (set *IntSet) AsUints() []uint {
	ids := make([]uint, 0, set.count)
	for ok, id := set.GetFirstID(); ok; ok, id = set.GetNextID(id) {
		ids = append(ids, id)
	}
	return ids
}

func (set *IntSet) CountMembers() uint {
	count := 0
	for i := set.start; i <= set.end; i++ {
		count += bits.OnesCount64(set.vs[i])
	}
	set.count = uint(count)
	return set.count
}

//Size gets an upper bound on the size of this set. After a call to CountMembers this value
//is accurate until elements are removed.
func (set *IntSet) Size() uint {
	return set.count
}

func (set *IntSet) String() string {
	s := "{"
	first := true
	for ok, v := set.GetFirstID(); ok; ok, v = set.GetNextID(v) {
		if first {
			first = false
			s = fmt.Sprint(s, v)
		} else {
			s = fmt.Sprint(s, ",", v)
		}
	}
	return s + "}"
}
