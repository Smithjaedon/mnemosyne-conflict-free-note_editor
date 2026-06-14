package server

import (
	"testing"
)

func apply(doc string, op Op) string {
	return doc[:op.End] + op.Text + doc[op.End+op.DeleteCount:]
}

func TestApplyInsert(t *testing.T) {
	got := apply("hello", Op{End: 2, Text: "XY"})
	want := "heXYllo"
	if got != want {
		t.Errorf("insert at 2: got %q, want %q", got, want)
	}
}

func TestApplyAppend(t *testing.T) {
	got := apply("hello", Op{End: 5, Text: "s"})
	want := "hellos"
	if got != want {
		t.Errorf("append: got %q, want %q", got, want)
	}
}

func TestApplyPrepend(t *testing.T) {
	got := apply("hello", Op{End: 0, Text: "A"})
	want := "Ahello"
	if got != want {
		t.Errorf("prepend: got %q, want %q", got, want)
	}
}

func TestApplyDelete(t *testing.T) {
	got := apply("hello", Op{End: 2, DeleteCount: 1})
	want := "helo"
	if got != want {
		t.Errorf("delete at 2: got %q, want %q", got, want)
	}
}

func TestApplyReplace(t *testing.T) {
	got := apply("hello", Op{End: 2, Text: "XYZ", DeleteCount: 2})
	want := "heXYZo"
	if got != want {
		t.Errorf("replace at 2: got %q, want %q", got, want)
	}
}

func TestApplyDeleteEnd(t *testing.T) {
	got := apply("hello", Op{End: 4, DeleteCount: 1})
	want := "hell"
	if got != want {
		t.Errorf("delete last char: got %q, want %q", got, want)
	}
}

func TestApplyInsertEmpty(t *testing.T) {
	got := apply("hello", Op{End: 3, Text: ""})
	want := "hello"
	if got != want {
		t.Errorf("empty insert: got %q, want %q", got, want)
	}
}
