package apono_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// AccessRequestDetails represents the 'access_request_details' object in the request.
type CreateAccessRequestDetails struct {
	AccessBundleIDs []string `json:"access_bundle_ids,omitempty"` // IDs of access bundles
	ResourceIDs     []string `json:"resource_ids,omitempty"`      // IDs of specific resources
	Purpose         string   `json:"purpose,omitempty"`           // Short description of purpose
	ExpirationInSec int      `json:"expiration_in_sec,omitempty"` // Duration of access in seconds
	Justification   string   `json:"justification,omitempty"`     // Detailed justification
	AccountID       string   `json:"account_id,omitempty"`        // Specific account ID
	IntegrationsID  string   `json:"integration_id,omitempty"`    // ID of integration
	RequestUsersIDs []string `json:"request_users_ids,omitempty"` // IDs of users requesting access
	UserID          string   `json:"user_id,omitempty"`           // IDs of users requesting access
	Permissions     []string `json:"permissions,omitempty"`       // IDs of users requesting access
}

// AccessRequestResponse represents the full response body from a successful access request creation.
type AccessRequestResponse struct {
	ID                         string                     `json:"id"`
	Status                     string                     `json:"status"`
	StatusDetails              string                     `json:"status_details"`
	CreatedAtInSec             int64                      `json:"created_at_in_sec"`
	UpdatedAtInSec             int64                      `json:"updated_at_in_sec"`
	CreateAccessRequestDetails CreateAccessRequestDetails `json:"access_request_details"` // Reusing the request struct for details
	ReviewerID                 string                     `json:"reviewer_id,omitempty"`
	ReviewersIDs               []string                   `json:"reviewers_ids,omitempty"`
	JustificationForEmail      string                     `json:"justification_for_email,omitempty"`
}

type AccessRequestItem struct {
	ID             string            `json:"id"`
	Status         string            `json:"status"`
	DurationInSec  int               `json:"duration_in_sec"`
	Justification  string            `json:"justification"`
	CreationDate   string            `json:"creation_date"`
	RevocationDate string            `json:"revocation_date"`
	CustomFields   map[string]string `json:"custom_fields,omitempty"` // Use map[string]string for key-value pairs
	AccessGroups   []struct {        // Anonymous struct for AccessGroups as per example
		Integration struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"integration"`
		ResourceTypes []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"resource_types"`
	} `json:"access_groups,omitempty"`
	Bundle *struct { // Use pointer to struct for Bundle to allow it to be nil if not present
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"bundle,omitempty"`
}

type AccessRequestsResponse struct {
	Items      []AccessRequestItem `json:"items"`
	Pagination struct {
		NextPageToken string `json:"next_page_token,omitempty"` // Token for fetching the next page
	} `json:"pagination"`
}

// EntitlementItem represents a single entitlement item in the response.
type EntitlementItem struct {
	Integration struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"integration"`
	Resource struct {
		ID       string `json:"id"`
		SourceID string `json:"source_id"` // Note: documentation says "source_id", not "sourceId"
		Type     struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"type"`
		Name string `json:"name"`
	} `json:"resource"`
	Permission struct {
		Name string `json:"name"`
	} `json:"permission"`
	Status string `json:"status"`
}

// AccessRequestEntitlementsResponse represents the full JSON response for entitlements.
type AccessRequestEntitlementsResponse struct {
	Items      []EntitlementItem `json:"items"`
	Pagination struct {
		NextPageToken string `json:"next_page_token,omitempty"`
	} `json:"pagination"`
}

type AponoAPI struct {
	Token   *string
	BaseURL *string
}

func AponoAPINew(apiToken string) (*AponoAPI, error) {
	BaseURL := "https://api.apono.io"
	aponoAPI := AponoAPI{Token: &apiToken, BaseURL: &BaseURL}

	return &aponoAPI, nil
}

// createAponoAccessRequest makes an authenticated POST request to create a new Apono Access Request.
func (aponoAPI *AponoAPI) post(jsonBody []byte, endpoint string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 4. Set standard and Authorization headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *aponoAPI.Token))

	client := &http.Client{Timeout: 30 * time.Second} // Set a timeout for the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d: %s (Response: %s)", resp.StatusCode, resp.Status, string(responseBytes))
	}

	return responseBytes, nil
}

func (aponoAPI *AponoAPI) get(params url.Values, endpoint string) ([]byte, error) {
	if len(params) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}
	log.Printf("Sending GET request to: %s", endpoint)

	// Create the HTTP request
	req, err := http.NewRequest(http.MethodGet, endpoint, nil) // GET request, no body needed
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set required headers
	req.Header.Set("Host", "api.apono.io") // As specified in the curl example
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *aponoAPI.Token))
	req.Header.Set("Accept", "*/*") // As specified in the curl example

	// Execute the request with a timeout
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close() // Ensure the response body is closed

	// Read the response body
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s (Response: %s)", resp.StatusCode, resp.Status, string(responseBytes))
	}
	return responseBytes, nil
}

// createAccessRequest makes an authenticated POST request to create a new Apono Access Request.
func (aponoAPI *AponoAPI) createAccessRequest(createAccessRequestDetails CreateAccessRequestDetails) (*AccessRequestResponse, error) {
	// 1. Construct the full API endpoint
	endpoint := fmt.Sprintf("%s/api/v3/access-requests", *aponoAPI.BaseURL)

	// 2. Prepare the request body
	jsonBody, err := json.Marshal(createAccessRequestDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	responseBytes, err := aponoAPI.post(jsonBody, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// 8. Decode the JSON response
	var apiResponse AccessRequestResponse
	err = json.Unmarshal(responseBytes, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w (Response: %s)", err, string(responseBytes))
	}

	return &apiResponse, nil
}

func (aponoAPI *AponoAPI) RequestAccess(UserID string, ResourceIdPermissionPairs [][]string, IntegrationID, Justification string) {
	//accessBundleIDs = []string{"bundle-id-123", "bundle-id-456"}

	for _, ResourcePermission := range ResourceIdPermissionPairs {
		createAccessRequestDetails := CreateAccessRequestDetails{
			ResourceIDs:     []string{ResourcePermission[0]},
			IntegrationsID:  IntegrationID, // Corresponding integration for the resource
			Purpose:         "sad",
			ExpirationInSec: 3600 * 12, // 1 hour
			Justification:   "sad",
			UserID:          UserID,
			Permissions:     []string{ResourcePermission[1]},
			// RequestUsersIDs: []string{"user-id-of-requester"}, // Optional, defaults to the API token's user
		}

		fmt.Println("Attempting to create a new Apono Access Request...")

		accessRequest, err := aponoAPI.createAccessRequest(createAccessRequestDetails)
		if err != nil {
			log.Fatalf("Error creating Apono Access Request: %v", err)
		}

		fmt.Println("Apono Access Request created successfully!")
		fmt.Printf("Request ID: %s\n", accessRequest.ID)
		fmt.Printf("Status: %s\n", accessRequest.Status)
		fmt.Printf("Created At: %s\n", time.Unix(accessRequest.CreatedAtInSec, 0).Format(time.RFC3339))
	}
}

func (aponoAPI *AponoAPI) ListAccessRequests(limit int) ([]AccessRequestItem, error) {
	if limit < 1 {
		limit = 100
	}

	pageToken := ""

	response, err := aponoAPI.ListAccessRequestsRaw(limit, pageToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data ListAccessRequestsRaw: %w", err)
	}
	apiResponse := response.Items

	for response.Pagination.NextPageToken != "" {
		pageToken = response.Pagination.NextPageToken
		response, err = aponoAPI.ListAccessRequestsRaw(limit, pageToken)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch data ListAccessRequestsRaw: %w", err)
		}
		log.Printf("Fetched %d access requests", len(response.Items))
		apiResponse = append(apiResponse, response.Items...)
	}

	return apiResponse, nil
}

func (aponoAPI *AponoAPI) ListAccessRequestsRaw(limit int, pageToken string) (*AccessRequestsResponse, error) {
	params := url.Values{}
	params.Add("limit", fmt.Sprintf("%d", limit))

	if pageToken != "" {
		params.Add("page_token", pageToken)
	}

	// Append query parameters to the URL
	endpoint := fmt.Sprintf("%s/api/user/v4/access-requests", *aponoAPI.BaseURL)
	responseBytes, err := aponoAPI.get(params, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w (Response: %s)", err, string(responseBytes))
	}
	// Decode the JSON response into our Go struct
	var apiResponse AccessRequestsResponse
	err = json.Unmarshal(responseBytes, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w (Response: %s)", err, string(responseBytes))
	}

	return &apiResponse, nil
}

func (aponoAPI *AponoAPI) GetAccessRequestEntitlements(limit int, requestID string) ([]EntitlementItem, error) {
	if limit < 1 {
		limit = 100
	}

	pageToken := ""

	response, err := aponoAPI.GetAccessRequestEntitlementsRaw(limit, pageToken, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data GetAccessRequestEntitlementsRaw: %w", err)
	}
	apiResponse := response.Items

	for response.Pagination.NextPageToken != "" {
		pageToken = response.Pagination.NextPageToken
		response, err = aponoAPI.GetAccessRequestEntitlementsRaw(limit, pageToken, requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch data GetAccessRequestEntitlementsRaw: %w", err)
		}
		log.Printf("Fetched %d access request Entitlements", len(response.Items))
		apiResponse = append(apiResponse, response.Items...)
	}

	return apiResponse, nil
}

func (aponoAPI *AponoAPI) GetAccessRequestEntitlementsRaw(limit int, pageToken, requestID string) (*AccessRequestEntitlementsResponse, error) {
	params := url.Values{}
	params.Add("limit", fmt.Sprintf("%d", limit))

	if pageToken != "" {
		params.Add("page_token", pageToken)
	}

	// Append query parameters to the URL
	endpoint := fmt.Sprintf("%s/api/user/v4/access-requests/%s/entitlements", *aponoAPI.BaseURL, requestID)
	responseBytes, err := aponoAPI.get(params, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w (Response: %s)", err, string(responseBytes))
	}
	// Decode the JSON response into our Go struct
	var apiResponse AccessRequestEntitlementsResponse
	err = json.Unmarshal(responseBytes, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w (Response: %s)", err, string(responseBytes))
	}

	return &apiResponse, nil
}
