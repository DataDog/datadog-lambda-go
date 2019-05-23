package metrics

import (
	"fmt"
	"net/http"
)

type (
	// APIClient sends metrics to Datadog
	APIClient struct {
		apiKey     string
		appKey     string
		baseAPIURL string
		httpClient *http.Client
	}
)

// MakeAPIClient creates a new API client with the given api and app keys
func MakeAPIClient(baseAPIURL, apiKey, appKey string) *APIClient {
	httpClient := &http.Client{}
	return &APIClient{
		apiKey,
		appKey,
		baseAPIURL,
		httpClient,
	}
}

// PrewarmConnection sends a redundant GET request to the Datadog API to prewarm the TSL connection
func (cl *APIClient) PrewarmConnection() error {
	req, err := http.NewRequest("GET", cl.makeRoute("validate"), nil)
	if err != nil {
		return fmt.Errorf("Couldn't create prewarming request: %v", err)
	}
	cl.addAPICredentials(req)
	_, err = cl.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Couldn't contact server for prewarm request %v", err)
	}
	return nil
}

func (cl *APIClient) addAPICredentials(req *http.Request) {
	query := req.URL.Query()
	query.Add(apiKeyParam, cl.apiKey)
	query.Add(appKeyParam, cl.appKey)
	req.URL.RawQuery = query.Encode()
}

func (cl *APIClient) makeRoute(route string) string {
	return fmt.Sprintf("%s/%s", cl.baseAPIURL, route)
}
