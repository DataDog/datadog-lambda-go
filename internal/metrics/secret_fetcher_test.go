// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.
package metrics

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/stretchr/testify/assert"
)

type mockSecretsManagerClient struct {
	secretsmanageriface.SecretsManagerAPI

	secretString string
	secretBinary []byte
}

func (c mockSecretsManagerClient) GetSecretValue(
	input *secretsmanager.GetSecretValueInput,
) (*secretsmanager.GetSecretValueOutput, error) {
	out := &secretsmanager.GetSecretValueOutput{SecretBinary: c.secretBinary}
	if c.secretString != "" {
		out.SecretString = &c.secretString
	}
	return out, nil
}

func TestSecretsManagerSecretsFetcher(t *testing.T) {
	tests := map[string]struct {
		client mockSecretsManagerClient
		want   string
	}{
		"secret is string": {
			client: mockSecretsManagerClient{secretString: "3333333333"},
			want:   "3333333333",
		},
		"secret is binary": {
			client: mockSecretsManagerClient{secretBinary: []byte("4444444444")},
			want:   "4444444444",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fetcher := &secretsManagerSecretFether{client: tt.client}
			got, err := fetcher.FetchSecret(
				"arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
