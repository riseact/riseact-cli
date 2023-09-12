package organizations

import (
	"github.com/AlecAivazis/survey/v2"
)

func PickOrganizationForm() (*Organization, error) {
	var organizationId int

	organizations, err := OrganizationSearch()

	if err != nil {
		return nil, err
	}

	options := make([]string, len(organizations))

	for i, org := range organizations {
		options[i] = org.Name
	}

	prompt := &survey.Select{
		Message: "Select an organization:",
		Options: options,
		Description: func(v string, i int) string {
			return organizations[i].Name
		},
	}

	survey.AskOne(prompt, &organizationId)

	return &organizations[organizationId], nil
}
