package main

import (
	"fmt"
	"strings"
)

const (
	And = "AND"
	Or  = "OR"
)

type Expr struct {
	// either bool expression
	op          string
	left, right *Expr

	// or just a string literal
	lit string
}

func (e *Expr) String() string {
	if e.lit != "" {
		return e.lit
	}
	return "(" + e.left.String() + " " + e.op + " " + e.right.String() + ")"
}

func parseQuery(s string) *Expr {
	parts := splitAtOps(s)
	if len(parts) == 1 {
		return &Expr{
			lit: strings.Trim(parts[0], "()"),
		}
	}
	return buildQueryTree(parts)
}

func buildQueryTree(parts []string) *Expr {
	switch len(parts) {
	case 0:
		panic("empty query!")
	case 1:
		return parseQuery(strings.Trim(parts[0], "()"))
	case 2:
		fmt.Printf("%#v\n", parts)
		panic("query can't have 2 parts!")
	}

	// try AND
	nextAnd := next(And, parts)
	if nextAnd != -1 {
		left := buildQueryTree(parts[:nextAnd])
		right := buildQueryTree(parts[nextAnd+1:])
		return &Expr{
			op:    And,
			left:  left,
			right: right,
		}
	}

	// try OR
	nextOr := next(Or, parts)
	if nextOr != -1 {
		left := buildQueryTree(parts[:nextOr])
		right := buildQueryTree(parts[nextOr+1:])
		return &Expr{
			op:    Or,
			left:  left,
			right: right,
		}
	}

	panic("multi-part query without operator!")
}

func splitAtOps(s string) []string {
	parts := []string{}

	lenOp, offset := nextOp(s)
	for offset != -1 {
		if offset == 0 {
			panic("operator at start of string!")
		}
		prefix := strings.TrimSpace(s[:offset])
		parts = append(parts, prefix)

		op := strings.TrimSpace(s[offset : offset+lenOp])
		parts = append(parts, op)

		s = s[offset+lenOp:]
		lenOp, offset = nextOp(s)
	}
	if s != "" {
		parts = append(parts, strings.TrimSpace(s))
	}
	return parts
}

func nextOp(s string) (int, int) {
	pos, depth := 0, 0
	for ; pos < len(s) && depth >= 0; pos++ {
		switch s[pos] {
		case '(':
			depth++
			continue
		case ')':
			depth--
			continue
		}
		if depth != 0 {
			continue
		}
		if strings.HasPrefix(s[pos:], "AND") {
			return len("AND"), pos
		} else if strings.HasPrefix(s[pos:], "OR") {
			return len("OR"), pos
		}
	}
	return 0, -1
}

func next(needle string, haystack []string) int {
	for i, blade := range haystack {
		if blade == needle {
			return i
		}
	}
	return -1
}
