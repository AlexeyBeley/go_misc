package azure_devops_api

import (
	"context"
	"log"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/work"
)

type WorkClient struct {
	Client        work.Client
	Configuration *Configuration
}

func WorkClientNew(Configuration *Configuration, context context.Context, connection *azuredevops.Connection) (*WorkClient, error) {

	workClient, err := work.NewClient(context, connection)
	if err != nil {
		log.Fatalf("Failed to create Git client: %v", err)
		return nil, err
	}
	ret := &WorkClient{Configuration: Configuration, Client: workClient}

	return ret, nil
}

func (workClient *WorkClient) GetIterations() ([]work.TeamSettingsIteration, error) {

	args := work.GetTeamIterationsArgs{Project: &workClient.Configuration.ProjectName}

	// Make the API call to get a page of repositories
	iters, err := workClient.Client.GetTeamIterations(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return *iters, nil
}
