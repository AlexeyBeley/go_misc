package azure_devops_api

import (
	"context"
	"log"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
)

type CoreClient struct {
	Client        core.Client
	Configuration *Configuration
}

func CoreClientNew(Configuration *Configuration, context context.Context, connection *azuredevops.Connection) (*CoreClient, error) {

	coreClient, err := core.NewClient(context, connection)
	if err != nil {
		log.Fatalf("Failed to create Git client: %v", err)
		return nil, err
	}
	ret := &CoreClient{Configuration: Configuration, Client: coreClient}

	return ret, nil
}

func (coreClient *CoreClient) GetTeams() ([]core.WebApiTeam, error) {

	args := core.GetAllTeamsArgs{}

	// Make the API call to get a page of repositories
	teams, err := coreClient.Client.GetAllTeams(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return *teams, nil
}

func (coreClient *CoreClient) GetProjects() ([]core.TeamProjectReference, error) {

	args := core.GetProjectsArgs{}

	// Make the API call to get a page of repositories
	GetProjectsResponseValue, err := coreClient.Client.GetProjects(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return GetProjectsResponseValue.Value, nil
}
