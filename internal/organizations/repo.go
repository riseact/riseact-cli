package organizations

import (
	"context"
	"riseact/internal/gql"
)

type Organization struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func OrganizationSearch() ([]Organization, error) {
	client, err := gql.GetClient()

	if err != nil {
		return nil, err
	}

	pagination := gql.PaginationInput{First: 1000}
	resp, err := gql.OrganizationSearch(context.Background(), *client, pagination)

	if err != nil {
		return nil, err
	}

	organizations := make([]Organization, len(resp.Organizations.Edges))

	for i, org := range resp.Organizations.Edges {
		organizations[i] = Organization{
			Id:   org.Node.Id,
			Name: org.Node.Name,
		}
	}

	return organizations, nil

}
