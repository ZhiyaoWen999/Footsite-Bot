package core

import (
	"os"

	"github.com/gocarina/gocsv"
)

type Profile struct {
	ProfileName string `csv:"PROFILES_NAME"`
	FirstName   string `csv:"FIRST_NAME"`
	LastName    string `csv:"LAST_NAME"`
	Email       string `csv:"EMAIL"`
	Phone       string `csv:"PHONE_NUMBER"`
	State       string `csv:"STATE"`
	City        string `csv:"CITY"`
	Line1       string `csv:"ADDRESS_LINE_1"`
	Line2       string `csv:"ADDRESS_LINE_2"`
	PostalCode  string `csv:"ZIP"`
	CCNumber    string `csv:"CARD_NUMBER"`
	ExpMonth    string `csv:"EXPIRE_MONTH"`
	ExpYear     string `csv:"EXPIRE_YEAR"`
	Cvv         string `csv:"CARD_CVV"`
}

func LoadProfile(filePath string) ([]Profile, error) {
	f, _ := os.Open(filePath)

	defer f.Close()

	profiles := []Profile{}

	err := gocsv.UnmarshalFile(f, &profiles)

	return profiles, err

}

// A handy map of US state codes to full names
var usc = map[string]string{
	"AL": "Alabama",
	"AK": "Alaska",
	"AZ": "Arizona",
	"AR": "Arkansas",
	"CA": "California",
	"CO": "Colorado",
	"CT": "Connecticut",
	"DE": "Delaware",
	"FL": "Florida",
	"GA": "Georgia",
	"HI": "Hawaii",
	"ID": "Idaho",
	"IL": "Illinois",
	"IN": "Indiana",
	"IA": "Iowa",
	"KS": "Kansas",
	"KY": "Kentucky",
	"LA": "Louisiana",
	"ME": "Maine",
	"MD": "Maryland",
	"MA": "Massachusetts",
	"MI": "Michigan",
	"MN": "Minnesota",
	"MS": "Mississippi",
	"MO": "Missouri",
	"MT": "Montana",
	"NE": "Nebraska",
	"NV": "Nevada",
	"NH": "New Hampshire",
	"NJ": "New Jersey",
	"NM": "New Mexico",
	"NY": "New York",
	"NC": "North Carolina",
	"ND": "North Dakota",
	"OH": "Ohio",
	"OK": "Oklahoma",
	"OR": "Oregon",
	"PA": "Pennsylvania",
	"RI": "Rhode Island",
	"SC": "South Carolina",
	"SD": "South Dakota",
	"TN": "Tennessee",
	"TX": "Texas",
	"UT": "Utah",
	"VT": "Vermont",
	"VA": "Virginia",
	"WA": "Washington",
	"WV": "West Virginia",
	"WI": "Wisconsin",
	"WY": "Wyoming",
	// Territories
	"AS": "American Samoa",
	"DC": "District of Columbia",
	"FM": "Federated States of Micronesia",
	"GU": "Guam",
	"MH": "Marshall Islands",
	"MP": "Northern Mariana Islands",
	"PW": "Palau",
	"PR": "Puerto Rico",
	"VI": "Virgin Islands",
	// Armed Forces (AE includes Europe, Africa, Canada, and the Middle East)
	"AA": "Armed Forces Americas",
	"AE": "Armed Forces Europe",
	"AP": "Armed Forces Pacific",
}

func makekey(m map[string]string, value string) (key string, ok bool) {
	for k, v := range m {
		if v == value {
			key = k
			ok = true
			return
		}
	}
	return
}
