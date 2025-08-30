package azure_devops_api

import (
	"context"
	"fmt"
	"log"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/graph"
)

type GraphClient struct {
	Client        graph.Client
	Configuration *Configuration
}

func GraphClientNew(Configuration *Configuration, context context.Context, connection *azuredevops.Connection) (*GraphClient, error) {

	graphClient, err := graph.NewClient(context, connection)
	if err != nil {
		log.Fatalf("Failed to create Git client: %v", err)
		return nil, err
	}
	ret := &GraphClient{Configuration: Configuration, Client: graphClient}

	return ret, nil
}

func (graphClient *GraphClient) ListUsers() ([]graph.GraphUser, error) {

	args := graph.ListUsersArgs{}

	// Make the API call to get a page of repositories
	response, err := graphClient.Client.ListUsers(context.Background(), args)
	if err != nil {
		return nil, err
	}

	// Check if any repositories were returned
	if *response.GraphUsers == nil || len(*response.GraphUsers) == 0 {
		return nil, fmt.Errorf("fetched  %v", *response.GraphUsers)
	}

	return *response.GraphUsers, nil
}

func (graphClient *GraphClient) GetUser(UserDescriptor *string) (*graph.GraphUser, error) {

	args := graph.GetUserArgs{UserDescriptor: UserDescriptor}

	// Make the API call to get a page of repositories
	response, err := graphClient.Client.GetUser(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return response, nil
}
