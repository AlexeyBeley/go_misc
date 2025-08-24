package azure_devops_api

import (
	"context"
	"fmt"
	"log"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type BuildClient struct {
	Client        build.Client
	Configuration *Configuration
}

func BuildClientNew(Configuration *Configuration, context context.Context, connection *azuredevops.Connection) (*BuildClient, error) {

	buildClient, err := build.NewClient(context, connection)
	if err != nil {
		log.Fatalf("Failed to create Build client: %v", err)
	}
	ret := &BuildClient{Configuration: Configuration, Client: buildClient}

	return ret, nil
}

func (buildClient *BuildClient) GetDefinitions() ([]build.BuildDefinitionReference, error) {

	args := build.GetDefinitionsArgs{
		Project: &buildClient.Configuration.ProjectName,
	}

	// Make the API call to get a page of repositories
	BuildDefinitionReferences, err := buildClient.Client.GetDefinitions(context.Background(), args)
	if err != nil {
		return nil, err
	}

	// Check if any repositories were returned
	if BuildDefinitionReferences == nil || len(BuildDefinitionReferences.Value) == 0 {
		return nil, fmt.Errorf("fetched  %v", BuildDefinitionReferences)
	}

	return BuildDefinitionReferences.Value, nil
}

func (buildClient *BuildClient) GetDefinition(DefinitionId *int) (*build.BuildDefinition, error) {

	args := build.GetDefinitionArgs{
		Project:      &buildClient.Configuration.ProjectName,
		DefinitionId: DefinitionId,
	}

	// Make the API call to get a page of repositories
	pipeline, err := buildClient.Client.GetDefinition(context.Background(), args)
	if err != nil {
		return nil, err
	}

	return pipeline, nil
}
