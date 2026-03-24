package wtl

import "strings"

type policy interface {
	Initialize(prompt string) policyState
	AfterTurn(policyState, turnOutcome) policyDecision
	OnWaitResolved(policyState, string) policyDecision
}

type simpleLoopPolicy struct{}

func (simpleLoopPolicy) Initialize(prompt string) policyState {
	return policyState{
		Runnable:  true,
		Directive: directiveContinue,
		Plan: executionPlan{
			Phase:      "",
			Prompt:     prompt,
			ThreadMode: threadModeReuse,
		},
	}
}

func (simpleLoopPolicy) AfterTurn(state policyState, outcome turnOutcome) policyDecision {
	next := directiveContinue
	if outcome.Err != nil || outcome.Status == "failed" {
		next = directiveRetry
	} else if strings.Contains(outcome.Response, completionMarker) {
		next = directiveComplete
	}

	nextState := state
	nextState.Directive = next
	nextState.Waiting = false
	nextState.WaitReason = ""
	nextState.Terminal = next == directiveComplete
	nextState.Runnable = !nextState.Terminal
	if next == directiveWait {
		nextState.Waiting = true
		nextState.Runnable = false
	}
	return policyDecision{Directive: next, State: nextState}
}

func (simpleLoopPolicy) OnWaitResolved(state policyState, _ string) policyDecision {
	state.Waiting = false
	state.WaitReason = ""
	state.Runnable = !state.Terminal
	state.Directive = directiveContinue
	return policyDecision{Directive: directiveContinue, State: state}
}
