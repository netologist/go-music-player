package musicplayer

import (
	"iter"
	"sync"
)

// LockedSeq2 returns an iter.Seq2 that iterates over a slice with index and element
// under the protection of a sync.Locker. The lock is released safely when the loop
// completes or exits early (short-circuit / break). Zero extra allocations (0-alloc).
func LockedSeq2[T any](locker sync.Locker, getItems func() []T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		locker.Lock()
		defer locker.Unlock()
		items := getItems()
		for i, item := range items {
			if !yield(i, item) {
				return
			}
		}
	}
}

// LockedSeq returns an iter.Seq that iterates over a slice (element only)
// under the protection of a sync.Locker. The lock is released safely when the loop
// completes or exits early (short-circuit / break). Zero extra allocations (0-alloc).
func LockedSeq[T any](locker sync.Locker, getItems func() []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		locker.Lock()
		defer locker.Unlock()
		items := getItems()
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
}

// SliceSeq2 returns a standard iter.Seq2 that produces index and element for
// each entry of the given slice.
func SliceSeq2[T any](items []T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, item := range items {
			if !yield(i, item) {
				return
			}
		}
	}
}

// SliceSeq returns a standard iter.Seq that produces each element of the given slice.
func SliceSeq[T any](items []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
}

// FilterSeq wraps an iter.Seq with a predicate, returning a lazy iter.Seq that
// only yields elements for which the predicate returns true.
func FilterSeq[T any](seq iter.Seq[T], predicate func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range seq {
			if predicate(item) {
				if !yield(item) {
					return
				}
			}
		}
	}
}

// Filter materialises an iter.Seq into a slice, keeping only elements for which
// the predicate returns true.
func Filter[T any](seq iter.Seq[T], predicate func(T) bool) []T {
	var out []T
	for item := range seq {
		if predicate(item) {
			out = append(out, item)
		}
	}
	return out
}

// ForEach iterates over an iter.Seq2 in order.
// The loop exits early if fn returns false (short-circuiting).
func ForEach[T any](seq iter.Seq2[int, T], fn func(int, T) bool) {
	for i, item := range seq {
		if !fn(i, item) {
			break
		}
	}
}

// Map returns a lazy iter.Seq that applies transform to each element of seq.
func Map[T, U any](seq iter.Seq[T], transform func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for item := range seq {
			if !yield(transform(item)) {
				return
			}
		}
	}
}

// Collect materialises all elements of an iter.Seq into a slice.
func Collect[T any](seq iter.Seq[T]) []T {
	var out []T
	for item := range seq {
		out = append(out, item)
	}
	return out
}

// Find returns the first element that satisfies the predicate and true.
// Returns the zero value and false if no match is found.
func Find[T any](seq iter.Seq[T], predicate func(T) bool) (T, bool) {
	for item := range seq {
		if predicate(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}
