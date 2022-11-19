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
		return nil, fmt.Errorf("empty input")
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
