// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.
package metrics

import (
	"errors"
	"fmt"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

// SecretFetcher attempts to fetch a secret.
type SecretFetcher interface {
	FetchSecret(secretID string) (string, error)
}

// secretsManagerSecretFether fetches a secret from AWS Secrets Manager.
type secretsManagerSecretFether struct {
	client secretsmanageriface.SecretsManagerAPI
}

// MakeSecretsManagerSecretFetcher creates a new SecretFetcher which uses the AWS
// Secrets Manager service to fetch a secret.
func MakeSecretsManagerSecretFetcher() SecretFetcher {
	return &secretsManagerSecretFether{
		client: secretsmanager.New(session.Must(session.NewSession())),
	}
}

// FetchSecret fetches and returns a secret for a given secret ID.
func (sf *secretsManagerSecretFether) FetchSecret(secretID string) (string, error) {
	logger.Debug("Fetching Secrets Manager secret " + secretID)
	output, err := sf.client.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		return "", fmt.Errorf(
			"could not retrieve Secrets Manager secret %q: %s", secretID, err,
		)
	}

	if s := output.SecretString; s != nil {
		return *s, nil
	}

	if b := output.SecretBinary; b != nil {
		// SecretBinary field has been automatically base64 decoded by the AWS SDK
		return string(b), nil
	}

	// Should not happen but let's handle this gracefully
	logger.Error(errors.New(
		"Secrets Manager returned something but there seems to be no data available",
	))
	return "", nil
}
