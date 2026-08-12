package azure_devops_api

import (
	"context"
	"log"

	"github.com/google/uuid"
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

func (workClient *WorkClient) GetTeamIterations(teamId *string) ([]work.TeamSettingsIteration, error) {
	args := work.GetTeamIterationsArgs{Project: &workClient.Configuration.ProjectName, Team: teamId}

	// Make the API call to get a page of repositories
	iters, err := workClient.Client.GetTeamIterations(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return *iters, nil
}

func (workClient *WorkClient) GetTeamFieldValues(teamName *string) (*work.TeamFieldValues, error) {

	args := work.GetTeamFieldValuesArgs{Project: &workClient.Configuration.ProjectName, Team: teamName}

	response, err := workClient.Client.GetTeamFieldValues(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Project *string
// (required) ID of the iteration
// IterationId *uuid.UUID
// (optional) Team ID or team name
// Team *string
func (workClient *WorkClient) GetIterationWorkItems(teamName *string, iterationId *uuid.UUID) (*work.IterationWorkItems, error) {

	args := work.GetIterationWorkItemsArgs{Project: &workClient.Configuration.ProjectName, Team: teamName, IterationId: iterationId}

	response, err := workClient.Client.GetIterationWorkItems(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return response, nil
}
