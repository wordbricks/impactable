package wtl

import "strings"

type policy interface {
	InitialPhase() string
	Next(turnOutcome) directive
}

type simpleLoopPolicy struct{}

func (simpleLoopPolicy) InitialPhase() string {
	return ""
}

func (simpleLoopPolicy) Next(outcome turnOutcome) directive {
	if outcome.Err != nil || outcome.Status == "failed" {
		return directiveRetry
	}
	if strings.Contains(outcome.Response, completionMarker) {
		return directiveComplete
	}
	return directiveContinue
}
