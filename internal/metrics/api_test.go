package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mockAPIKey = "12345"
	mockAppKey = "678910"
)

func TestAddAPICredentials(t *testing.T) {
	cl := MakeAPIClient("", mockAPIKey, mockAppKey)
	req, _ := http.NewRequest("GET", "http://some-api.com/endpoint", nil)
	cl.addAPICredentials(req)
	assert.Equal(t, "http://some-api.com/endpoint?api_key=12345&application_key=678910", req.URL.String())
}

func TestPrewarmConnection(t *testing.T) {

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "/validate?api_key=12345&application_key=678910", r.URL.String())
	}))
	defer server.Close()

	cl := MakeAPIClient(server.URL, mockAPIKey, mockAppKey)
	err := cl.PrewarmConnection()

	assert.NoError(t, err)
	assert.True(t, called)
}
