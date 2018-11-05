package parser

import (
	"strings"
	"fmt"
	"text/scanner"
	"github.com/revel/cmd/controller"
)

type status int

const (
	initial status = iota
	annotationName
	dataName
	dataValue
	done
)

// Parses the annotation for this line, returns an error if parsing failed
func parseAnnotation(line string) (controller.FunctionalAnnotation, error) {
	withoutComment := strings.TrimLeft(strings.TrimSpace(line), "/")

	annotation := controller.FunctionalAnnotation{
		Name:       "",
		Data: make(map[string]string),
	}

	var s scanner.Scanner
	s.Init(strings.NewReader(withoutComment))

	var tok rune
	currentStatus := initial
	var attrName string

	for tok != scanner.EOF && currentStatus < done {
		tok = s.Scan()
		switch tok {
		case '@':
			currentStatus = annotationName
		case '(':
			currentStatus = dataName
		case '=':
			currentStatus = dataValue
		case ',':
			currentStatus = dataName
		case ')':
			currentStatus = done
		case scanner.Ident:
			switch currentStatus {
			case annotationName:
				annotation.Name = s.TokenText()
			case dataName:
				attrName = s.TokenText()
			}
		default:
			switch currentStatus {
			case dataValue:
				annotation.Data[strings.ToLower(attrName)] = strings.Trim(s.TokenText(), "\"")
			}
		}
	}

	if currentStatus != done {
		return annotation, fmt.Errorf("Invalid completion-status %v for annotation:%s", currentStatus, line)
	}
	return annotation, nil
}
