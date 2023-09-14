package cohabitaters

import (
	"google.golang.org/api/people/v1"
)

type UserState struct {
	GoogleForceApproval  bool
	ContactGroups        []*people.ContactGroup
	SelectedResourceName string
}
