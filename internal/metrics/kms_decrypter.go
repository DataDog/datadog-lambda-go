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
	sess, err := session.NewSession(nil)
	if err != nil {
		logger.Error(fmt.Errorf("could not create a new aws-sdk session: %v", err))
		panic(err)
	}
	return &kmsDecrypter{
		kmsClient: kms.New(sess),
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
		return "", fmt.Errorf("failed to encode cipher text to base64: %v", err)
	}

	// When the API key is encrypted using the AWS console, the function name is added as an
	// encryption context. When the API key is encrypted using the AWS CLI, no encryption context
	// is added. We need to try decrypting the API key both with and without the encryption context.

	// Try without encryption context, in case API key was encrypted using the AWS CLI
	functionName := os.Getenv(functionNameEnvVar)
	params := &kms.DecryptInput{
		CiphertextBlob: decodedBytes,
	}
	response, err := kmsClient.Decrypt(params)

	if err != nil {
		logger.Debug("Failed to decrypt ciphertext without encryption context, retrying with encryption context")
		// Try with encryption context, in case API key was encrypted using the AWS Console
		params = &kms.DecryptInput{
			CiphertextBlob: decodedBytes,
			EncryptionContext: map[string]*string{
				encryptionContextKey: &functionName,
			},
		}
		response, err = kmsClient.Decrypt(params)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt ciphertext with kms: %v", err)
		}
	}

	plaintext := string(response.Plaintext)
	return plaintext, nil
}
