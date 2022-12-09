package cohabitaters

import (
	"fmt"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"google.golang.org/api/people/v1"
)

type XmasCard struct {
	Names   []string
	Address Address
}

type Address struct {
	StreetAddress  string
	StreetAddress2 string
	City           string
	Region         string
	Country        string
	PostalCode     string
}

func NewAddress(in *people.Address) Address {
	return Address{
		StreetAddress:  in.StreetAddress,
		StreetAddress2: in.ExtendedAddress,
		City:           in.City,
		Region:         in.Region,
		Country:        in.Country,
		PostalCode:     in.PostalCode,
	}
}

func PickHomeAddress(in []*people.Address) (*people.Address, error) {
	switch {
	case len(in) == 0:
		return nil, nil
	case len(in) == 1:
		return in[0], nil
	default:
		for _, addr := range in {
			if strings.ToLower(addr.Type) == "home" {
				return addr, nil
			}
		}
		return nil, fmt.Errorf("no home address")
	}
}

func FuzzyTrimMatch(a, b string) bool {
	return fuzzy.Match(strings.TrimSpace(a), strings.TrimSpace(b))
}

func FuzzyAddressMatch(a *people.Address, b Address) bool {
	//	fmt.Println(a.City, b.City, fuzzyTrimMatch(a.City, b.City))
	return FuzzyTrimMatch(a.City, b.City) &&
		FuzzyTrimMatch(a.StreetAddress, b.StreetAddress)
}

func GetXmasCards(svc *people.Service, contactGroupResourceName string) ([]XmasCard, error) {
	cgResp, err := svc.ContactGroups.Get(contactGroupResourceName).MaxMembers(1000).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve contactGroup members: %w", err)
	}
	if len(cgResp.MemberResourceNames) == 0 {
		return nil, fmt.Errorf("empty contactGroup members")
	}

	pplResp, err := svc.People.GetBatchGet().ResourceNames(cgResp.MemberResourceNames...).PersonFields("names,addresses").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve people: %w", err)
	}
	if len(pplResp.Responses) == 0 {
		return nil, fmt.Errorf("empty people responses")
	}

	var cards []XmasCard

	for _, pr := range pplResp.Responses {
		name := pr.Person.Names[0].DisplayName
		found := false
		homeAddr, err := PickHomeAddress(pr.Person.Addresses)
		if err != nil {
			return nil, fmt.Errorf("error picking home address for %s: %w", name, err)
		}
		if homeAddr == nil {
			continue // unable to pick home address
		}

		for idx, card := range cards {
			if FuzzyAddressMatch(homeAddr, card.Address) {
				cards[idx].Names = append(cards[idx].Names, name)
				found = true
			}
		}
		if !found {
			cards = append(cards, XmasCard{
				Names:   []string{name},
				Address: NewAddress(homeAddr),
			})
		}
	}

	return cards, nil
}
