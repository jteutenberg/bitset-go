# Efficient sets for unsigned integers

A memory and CPU efficient Golang implementation of sets of `uint` unsigned integers.

# When to use this?

In many cases using a hashmap-based (`map`-based) implementation is sufficient. However this implementation may be preferred if any of the following hold:

- you need to apply set-to-set operations like union and intersection
- in many cases your sets contain contiguous intervals

This implementation may not be suitable if your sets are sparse and span a large range e.g. millions of values.

# Details

Initially sets are represented as intervals. This is memory efficient and for many overlapping intervals also leads to very efficient union and intersection operations.

When an operation on a set splits its interval, its representation is switched to a bitset. These bitsets are backed by a slice of `uint64` spanning a range of values around the set -- so not necessarily starting at 0. This is to maintain memory efficiency in cases when a set contains a small range but with large values, e.g. the set `{1_000_000_000, 1_000_000_002}`.

Note that a bitset-backed set is never reverted to an interval representation, even if it has returned to containing a contiguous series of values.

The code manages all variations of interval-interval, interval-bitset, and bitset-interval operations.