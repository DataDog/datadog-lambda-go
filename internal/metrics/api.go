package metrics

import (
	"bytes"
	"encoding/json"
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

	postMetricsModel struct {
		Series []APIMetric `json:"series"`
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
		return fmt.Errorf("Couldn't contact server for prewarm request")
	}
	return nil
}

// SendMetrics posts a batch metrics payload to the Datadog API
func (cl *APIClient) SendMetrics(metrics []APIMetric) error {
	content, err := marshalAPIMetricsModel(metrics)
	if err != nil {
		return fmt.Errorf("Couldn't marshal metrics model: %v", err)
	}
	body := bytes.NewBuffer(content)

	req, err := http.NewRequest("POST", cl.makeRoute("series"), body)
	if err != nil {
		return fmt.Errorf("Couldn't create send metrics request:%v", err)
	}
	defer req.Body.Close()

	cl.addAPICredentials(req)

	resp, err := cl.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("Failed to send metrics to API")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Failed to send metrics to API. Status Code %d", resp.StatusCode)
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

func marshalAPIMetricsModel(metrics []APIMetric) ([]byte, error) {
	pm := postMetricsModel{}
	pm.Series = metrics
	return json.Marshal(pm)
}
