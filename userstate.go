package cohabitaters

import (
	"golang.org/x/oauth2"
	oauth2_api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/people/v1"
)

type UserState struct {
	GoogleForceApproval  bool
	Token                *oauth2.Token
	Userinfo             *oauth2_api.Userinfo
	ContactGroups        []*people.ContactGroup
	SelectedResourceName string
}
