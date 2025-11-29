package forms

import "link-mgmt-go/pkg/models"

type submitErrorMsg struct {
	err error
}

type submitSuccessMsg struct {
	link *models.Link
}

type linksLoadedMsg struct {
	links []models.Link
	err   error
}

type deleteErrorMsg struct {
	err error
}

type deleteSuccessMsg struct{}
