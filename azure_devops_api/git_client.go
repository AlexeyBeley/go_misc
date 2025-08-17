package azure_devops_api

import (
	"context"
	"fmt"
	"log"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type GitClient struct {
	Client        git.Client
	Configuration Configuration
}

func GitClientNew(Configuration Configuration, context context.Context, connection *azuredevops.Connection) (*GitClient, error) {

	gitClient, err := git.NewClient(context, connection)
	if err != nil {
		log.Fatalf("Failed to create Git client: %v", err)
	}
	ret := &GitClient{Configuration: Configuration, Client: gitClient}

	return ret, nil
}

func (gitClient *GitClient) GetRepositories() ([]git.GitRepository, error) {

	args := git.GetRepositoriesArgs{
		Project: &gitClient.Configuration.ProjectName,
	}

	// Make the API call to get a page of repositories
	repositories, err := gitClient.Client.GetRepositories(context.Background(), args)
	if err != nil {
		return nil, err
	}

	// Check if any repositories were returned
	if repositories == nil || len(*repositories) == 0 {
		return nil, fmt.Errorf("fetched  %v", repositories)
	}

	return *repositories, nil
}
