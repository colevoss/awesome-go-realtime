package server

import "testing"

func TestNewDotPath(t *testing.T) {
	dotPath1 := NewDotPath("test.{id}.channel")
	dotPath2 := NewDotPath("test.{id}.channel.{otherId}")

	if dotPath1.paramCount != 1 {
		t.Errorf("%s should have 1 param", dotPath1.path)
	}

	if dotPath2.paramCount != 2 {
		t.Errorf("%s should have 2 param", dotPath2.path)
	}
}

func TestNewDotPathNoParams(t *testing.T) {
	dotPath := NewDotPath("test.channel")

	if dotPath.paramCount != 0 {
		t.Errorf("%s should have 0 params", dotPath.path)
	}

	doesMatch, params := dotPath.doesMatch("test.channel")

	if !doesMatch {
		t.Errorf("test.channel should match dotPath")
	}

	if params.Len() != 0 {
		t.Error("params should have length of 0")
	}
}

func TestDotPathDoesMath(t *testing.T) {
	dotPath := NewDotPath("test.{id}.channel")

	testPath1 := "test.1.channel"
	if testMatch, _ := dotPath.doesMatch(testPath1); !testMatch {
		t.Errorf("%s should match path %s", testPath1, dotPath.path)
	}

	testPath2 := "test.channel"

	if testMatch, _ := dotPath.doesMatch(testPath2); testMatch {
		t.Errorf("%s should not match path %s", testPath2, dotPath.path)
	}
}

func TestDotPathParams(t *testing.T) {
	dotPath := NewDotPath("test.{id}.channel.{otherParam}")

	testPath1 := "test.1.channel.other"

	_, params := dotPath.doesMatch(testPath1)

	if p, ok := params.Get("id"); p != "1" || !ok {
		t.Errorf("id param should be %s", "1")
	}

	if p, ok := params.Get("otherParam"); p != "other" || !ok {
		t.Errorf("otherParam param should be %s", "other")
	}
}
