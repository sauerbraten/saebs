package main

import (
	"strings"
	"testing"
)

func TestParseQuery(t *testing.T) {
	input := "stutter AND (man OR boy)"
	expected := "(stutter AND (man OR boy))"
	e := parseQuery(input)

	if e.String() != expected {
		t.Fatal("want", expected, "but got", e.String())
	}

	// same, but wrapped in ()
	input = "(stutter AND (man OR boy))"
	expected = "(stutter AND (man OR boy))"
	e = parseQuery(input)

	if input != e.String() {
		t.Fatal("want", input, "but got", e.String())
	}

	// test precedence
	input = "stutter AND man OR boy"
	expected = "(stutter AND (man OR boy))"
	e = parseQuery(input)

	if e.String() != expected {
		t.Fatal("want", expected, "but got", e.String())
	}
}

func TestSplitAtOps(t *testing.T) {
	input := "stutter AND (man OR boy)"
	expected := "stutter AND (man OR boy)"
	parts := splitAtOps(input)
	joined := strings.Join(parts, " ")

	if joined != expected {
		t.Fatal("want", expected, "but got", joined)
	}

	// same, but wrapped in ()
	input = "(stutter AND (man OR boy))"
	expected = "stutter AND (man OR boy)"
	parts = splitAtOps(input)
	joined = strings.Join(parts, " ")

	if joined != expected {
		t.Fatal("want", expected, "but got", joined)
	}

	// same, but wrapped in ((()))
	input = "(((stutter AND (man OR boy))))"
	expected = "stutter AND (man OR boy)"
	parts = splitAtOps(input)
	joined = strings.Join(parts, " ")

	if joined != expected {
		t.Fatal("want", expected, "but got", joined)
	}
}
