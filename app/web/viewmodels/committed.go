package viewmodels

import "github.com/leomorpho/goship/app/web/ui"

type (
	DropdownIterable struct {
		ID     int `json:"id"`
		Object any `json:"object"`
	}

	CommittedModePageData struct {
		Friends        []DropdownIterable
		InvitationText string
		InvitationLink string
	}

	UpdateInAppModeForm struct {
		MatchProfileID int `form:"match_id" validate:"required"`
		Submission     ui.FormSubmission
	}
)
