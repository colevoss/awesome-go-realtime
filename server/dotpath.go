package server

import (
	"regexp"
	"strings"
)

type DotPath struct {
	path          string
	pathRegExpStr string
	paramCount    int
}

type Param struct {
	Key   string
	Value string
}

type Params []Param

func (p Params) Get(name string) (string, bool) {
	for _, entry := range p {
		if entry.Key == name {
			return entry.Value, true
		}
	}

	return "", false
}

func (p Params) Param(name string) (param string) {
	param, _ = p.Get(name)
	return
}

func (p Params) Len() int {
	return len(p)
}

var emptyParams = make(Params, 0)

func (dp *DotPath) doesMatch(path string) (bool, *Params) {
	if dp.paramCount == 0 {
		return path == dp.path, &emptyParams
	}

	pathRegExp := regexp.MustCompile(dp.pathRegExpStr)
	doesMatch := pathRegExp.MatchString(path)

	if !doesMatch {
		return doesMatch, nil
	}

	newParams := make(Params, dp.paramCount)

	matches := pathRegExp.FindStringSubmatch(path)

	for i, name := range pathRegExp.SubexpNames() {
		if i == 0 {
			continue
		}

		newParams[i-1] = Param{
			Key:   name,
			Value: matches[pathRegExp.SubexpIndex(name)],
		}
	}

	return doesMatch, &newParams
}

func NewDotPath(path string) *DotPath {
	pathRegExpStr, paramCount := parsePath(path)

	return &DotPath{path, pathRegExpStr, paramCount}
}

const delimiter = "."

func parsePath(path string) (string, int) {
	paramReg := regexp.MustCompile(`\{\w+\}`)

	parts := strings.Split(path, delimiter)

	pathRegString := ""

	var pathParts []string

	paramCount := 0

	for _, part := range parts {
		isParam := paramReg.MatchString(part)

		if isParam {
			pathParts = append(pathParts, getRegExpStrForParam(part))
			paramCount = paramCount + 1
		} else {
			pathParts = append(pathParts, part)
			pathRegString += part
		}
	}

	pathReg := strings.Join(pathParts, "\\.")

	return "^" + pathReg + "$", paramCount
}

func getRegExpStrForParam(param string) string {
	paramNameReg := regexp.MustCompile(`\w+`)

	paramName := paramNameReg.FindString(param)

	return "(?P<" + paramName + ">\\w+)"
}
