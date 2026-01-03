package managelinks

// Step constants for the manage links state machine
const (
	StepListLinks = iota
	StepActionMenu
	StepViewDetails
	StepDeleteConfirm
	StepScraping
	StepScrapeSaving
	StepScrapeDone
	StepDone
)

// DefaultWidth is the default terminal width fallback
const DefaultWidth = 80
