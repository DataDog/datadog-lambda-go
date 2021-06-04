/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */
package metrics

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
)

type (
	// Decrypter attempts to decrypt a key
	Decrypter interface {
		Decrypt(cipherText string) (string, error)
	}

	kmsDecrypter struct {
		kmsClient *kms.KMS
	}
)

// functionNameEnvVar is the environment variable that stores the Lambda function name
const functionNameEnvVar = "AWS_LAMBDA_FUNCTION_NAME"

// encryptionContextKey is the key added to the encryption context by the Lambda console UI
const encryptionContextKey = "LambdaFunctionName"

// MakeKMSDecrypter creates a new decrypter which uses the AWS KMS service to decrypt variables
func MakeKMSDecrypter() Decrypter {
	return &kmsDecrypter{
		kmsClient: kms.New(session.New(nil)),
	}
}

func (kd *kmsDecrypter) Decrypt(ciphertext string) (string, error) {
	return decryptKMS(kd.kmsClient, ciphertext)
}

// decryptKMS decodes and deciphers the base64-encoded ciphertext given as a parameter using KMS.
// For this to work properly, the Lambda function must have the appropriate IAM permissions.
func decryptKMS(kmsClient kmsiface.KMSAPI, ciphertext string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Failed to encode cipher text to base64: %v", err)
	}

	// The Lambda console UI changed the way it encrypts environment variables.
	// The current behavior as of May 2021 is to encrypt environment variables using the function name as an encryption context.
	// Previously, the behavior was to encrypt environment variables without an encryption context.
	// We need to try both, as supplying the incorrect encryption context will cause decryption to fail.

	// Try with encryption context
	functionName := os.Getenv(functionNameEnvVar)
	params := &kms.DecryptInput{
		CiphertextBlob: decodedBytes,
		EncryptionContext: map[string]*string{
			encryptionContextKey: &functionName,
		},
	}
	response, err := kmsClient.Decrypt(params)

	if err != nil {
		logger.Debug("Failed to decrypt ciphertext with encryption context, retrying without encryption context")
		// Try without encryption context
		params = &kms.DecryptInput{
			CiphertextBlob: decodedBytes,
		}
		response, err = kmsClient.Decrypt(params)
		if err != nil {
			return "", fmt.Errorf("Failed to decrypt ciphertext with kms: %v", err)
		}
	}

	plaintext := string(response.Plaintext)
	return plaintext, nil
}
