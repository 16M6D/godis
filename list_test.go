package main

import "testing"

func strListType() ListType {
	return ListType{
		EqualFunc: func(a, b *Gobj) bool {
			if a == nil || b == nil {
				return a == b
			}
			return a.StrVal() == b.StrVal()
		},
	}
}

func makeStrObj(s string) *Gobj {
	return CreateObject(GSTR, s)
}

func TestListCreate(t *testing.T) {
	l := ListCreate(strListType())
	if l.Length() != 0 {
		t.Fatalf("expected length 0, got %d", l.Length())
	}
	if l.First() != nil {
		t.Fatalf("expected nil first, got %v", l.First())
	}
	if l.Last() != nil {
		t.Fatalf("expected nil last, got %v", l.Last())
	}
}

func TestAppend(t *testing.T) {
	l := ListCreate(strListType())

	l.Append(makeStrObj("a"))
	if l.Length() != 1 {
		t.Fatalf("expected length 1, got %d", l.Length())
	}
	if l.First().Val.StrVal() != "a" {
		t.Fatalf("expected first 'a', got %q", l.First().Val.StrVal())
	}
	if l.Last().Val.StrVal() != "a" {
		t.Fatalf("expected last 'a', got %q", l.Last().Val.StrVal())
	}

	l.Append(makeStrObj("b"))
	l.Append(makeStrObj("c"))
	if l.Length() != 3 {
		t.Fatalf("expected length 3, got %d", l.Length())
	}
	if l.First().Val.StrVal() != "a" {
		t.Fatalf("expected first 'a', got %q", l.First().Val.StrVal())
	}
	if l.Last().Val.StrVal() != "c" {
		t.Fatalf("expected last 'c', got %q", l.Last().Val.StrVal())
	}

	// verify order: a -> b -> c
	n := l.First()
	expected := []string{"a", "b", "c"}
	for i, exp := range expected {
		if n == nil {
			t.Fatalf("node %d is nil", i)
		}
		if n.Val.StrVal() != exp {
			t.Fatalf("node %d expected %q, got %q", i, exp, n.Val.StrVal())
		}
		n = n.next
	}
}

func TestLPush(t *testing.T) {
	l := ListCreate(strListType())

	l.LPush(makeStrObj("a"))
	if l.Length() != 1 {
		t.Fatalf("expected length 1, got %d", l.Length())
	}
	if l.First().Val.StrVal() != "a" {
		t.Fatalf("expected first 'a', got %q", l.First().Val.StrVal())
	}

	l.LPush(makeStrObj("b"))
	l.LPush(makeStrObj("c"))
	if l.Length() != 3 {
		t.Fatalf("expected length 3, got %d", l.Length())
	}

	// order should be c -> b -> a
	n := l.First()
	expected := []string{"c", "b", "a"}
	for i, exp := range expected {
		if n == nil {
			t.Fatalf("node %d is nil", i)
		}
		if n.Val.StrVal() != exp {
			t.Fatalf("node %d expected %q, got %q", i, exp, n.Val.StrVal())
		}
		n = n.next
	}
}

func TestFind(t *testing.T) {
	l := ListCreate(strListType())

	if n := l.Find(makeStrObj("x")); n != nil {
		t.Fatalf("expected nil for find in empty list")
	}

	l.Append(makeStrObj("a"))
	l.Append(makeStrObj("b"))
	l.Append(makeStrObj("c"))

	n := l.Find(makeStrObj("b"))
	if n == nil || n.Val.StrVal() != "b" {
		t.Fatalf("expected to find 'b'")
	}

	if n := l.Find(makeStrObj("x")); n != nil {
		t.Fatalf("expected nil for non-existent value")
	}
}

func TestDelNodeHead(t *testing.T) {
	l := ListCreate(strListType())
	l.Append(makeStrObj("a"))
	l.Append(makeStrObj("b"))
	l.Append(makeStrObj("c"))

	l.DelNode(l.First())
	if l.Length() != 2 {
		t.Fatalf("expected length 2, got %d", l.Length())
	}
	if l.First().Val.StrVal() != "b" {
		t.Fatalf("expected first 'b', got %q", l.First().Val.StrVal())
	}
	if l.First().prev != nil {
		t.Fatalf("expected first.prev nil")
	}
}

func TestDelNodeTail(t *testing.T) {
	l := ListCreate(strListType())
	l.Append(makeStrObj("a"))
	l.Append(makeStrObj("b"))
	l.Append(makeStrObj("c"))

	l.DelNode(l.Last())
	if l.Length() != 2 {
		t.Fatalf("expected length 2, got %d", l.Length())
	}
	if l.Last().Val.StrVal() != "b" {
		t.Fatalf("expected last 'b', got %q", l.Last().Val.StrVal())
	}
	if l.Last().next != nil {
		t.Fatalf("expected last.next nil")
	}
}

func TestDelNodeMiddle(t *testing.T) {
	l := ListCreate(strListType())
	l.Append(makeStrObj("a"))
	l.Append(makeStrObj("b"))
	l.Append(makeStrObj("c"))

	middle := l.First().next // "b"
	l.DelNode(middle)
	if l.Length() != 2 {
		t.Fatalf("expected length 2, got %d", l.Length())
	}

	// order: a -> c
	if l.First().Val.StrVal() != "a" {
		t.Fatalf("expected first 'a', got %q", l.First().Val.StrVal())
	}
	if l.Last().Val.StrVal() != "c" {
		t.Fatalf("expected last 'c', got %q", l.Last().Val.StrVal())
	}
	if l.First().next != l.Last() {
		t.Fatalf("a.next should point to c")
	}
	if l.Last().prev != l.First() {
		t.Fatalf("c.prev should point to a")
	}
}

func TestDelNodeSingle(t *testing.T) {
	l := ListCreate(strListType())
	l.Append(makeStrObj("a"))

	l.DelNode(l.First())
	if l.Length() != 0 {
		t.Fatalf("expected length 0, got %d", l.Length())
	}
	if l.First() != nil {
		t.Fatalf("expected nil first")
	}
	if l.Last() != nil {
		t.Fatalf("expected nil last")
	}
}

func TestDelNodeNil(t *testing.T) {
	l := ListCreate(strListType())
	l.Append(makeStrObj("a"))

	l.DelNode(nil)
	if l.Length() != 1 {
		t.Fatalf("expected length unchanged (1), got %d", l.Length())
	}
}

func TestDelete(t *testing.T) {
	l := ListCreate(strListType())
	l.Append(makeStrObj("a"))
	l.Append(makeStrObj("b"))
	l.Append(makeStrObj("c"))

	l.Delete(makeStrObj("b"))
	if l.Length() != 2 {
		t.Fatalf("expected length 2, got %d", l.Length())
	}

	// deleting non-existent value
	l.Delete(makeStrObj("x"))
	if l.Length() != 2 {
		t.Fatalf("expected length unchanged (2), got %d", l.Length())
	}
}

func TestMixedAppendLPush(t *testing.T) {
	l := ListCreate(strListType())
	l.Append(makeStrObj("a"))   // a
	l.LPush(makeStrObj("b"))    // b -> a
	l.Append(makeStrObj("c"))   // b -> a -> c

	if l.Length() != 3 {
		t.Fatalf("expected length 3, got %d", l.Length())
	}

	n := l.First()
	expected := []string{"b", "a", "c"}
	for i, exp := range expected {
		if n == nil {
			t.Fatalf("node %d is nil", i)
		}
		if n.Val.StrVal() != exp {
			t.Fatalf("node %d expected %q, got %q", i, exp, n.Val.StrVal())
		}
		n = n.next
	}
}

func TestLength(t *testing.T) {
	l := ListCreate(strListType())
	if l.Length() != 0 {
		t.Fatalf("expected 0")
	}

	l.Append(makeStrObj("a"))
	l.Append(makeStrObj("b"))
	l.DelNode(l.First())
	if l.Length() != 1 {
		t.Fatalf("expected 1 after del, got %d", l.Length())
	}

	l.LPush(makeStrObj("c"))
	l.Delete(makeStrObj("b"))
	if l.Length() != 1 {
		t.Fatalf("expected 1, got %d", l.Length())
	}
}
