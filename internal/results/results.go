package results

import "strings"

type Result struct {
	ChecksPassing bool
	SkipCI        bool
	HoldPR        bool
	Messages      []string
}

func New() *Result {
	return &Result{
		ChecksPassing: true,
		SkipCI:        false,
		HoldPR:        false,
		Messages:      []string{},
	}
}

// AllowAccess returns true if checks are passing (or SkipCI is set) and the PR is not marked as hold
func (r *Result) AllowAccess() bool {
	return (r.ChecksPassing || r.SkipCI) && !r.HoldPR
}

func (r *Result) AddMessage(msg string) {
	r.Messages = append(r.Messages, msg)
}

func (r *Result) GetMessages() string {
	return strings.Join(r.Messages, "\n")
}
