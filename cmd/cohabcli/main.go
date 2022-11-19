package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	cohabitaters "github.com/bfallik/xmas-card-addresses"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, people.ContactsReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := people.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create people Client %v", err)
	}

	var resourceNameXmasCard string

	r1, err := srv.ContactGroups.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve contactGroups. %v", err)
	}
	for _, contactGroup := range r1.ContactGroups {
		if contactGroup.Name == "Xmas Card" {
			resourceNameXmasCard = contactGroup.ResourceName
		}
	}
	if len(resourceNameXmasCard) == 0 {
		log.Fatalf("No 'Xmas Card' contact group found.")
	}

	// contactGroups.get
	r2, err := srv.ContactGroups.Get(resourceNameXmasCard).MaxMembers(1000).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve contactGroup members. %v", err)
	}
	if len(r2.MemberResourceNames) == 0 {
		log.Fatalf("Empty contactGroup members. %v", err)
	}

	r3, err := srv.People.GetBatchGet().ResourceNames(r2.MemberResourceNames...).PersonFields("names,addresses").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve people. %v", err)
	}
	if len(r3.Responses) == 0 {
		log.Fatalf("Empty responses. %v", err)
	}

	var cards []cohabitaters.XmasCard

	for _, pr := range r3.Responses {
		name := pr.Person.Names[0].DisplayName
		found := false
		homeAddr, err := cohabitaters.PickHomeAddress(pr.Person.Addresses)
		if err != nil {
			log.Fatalf("Unable to pick home address for %s. %v", name, err)
		}

		for idx, card := range cards {
			if cohabitaters.FuzzyAddressMatch(homeAddr, card.Address) {
				cards[idx].Names = append(cards[idx].Names, name)
				found = true
			}
		}
		if !found {
			cards = append(cards, cohabitaters.XmasCard{
				Names:   []string{name},
				Address: cohabitaters.NewAddress(homeAddr),
			})
		}
	}

	for _, card := range cards {
		fmt.Printf("%v\n", card)
	}
}
