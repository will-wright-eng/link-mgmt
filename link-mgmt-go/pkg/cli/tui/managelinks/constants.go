package managelinks

// Step constants for the manage links state machine
const (
	StepListLinks = iota
	StepActionMenu
	StepViewDetails
	StepDeleteConfirm
	StepEnriching
	StepEnrichDone
	StepDone
)

// DefaultWidth is the default terminal width fallback
const DefaultWidth = 80
