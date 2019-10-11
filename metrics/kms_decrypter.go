/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */
package metrics

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
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

// MakeKMSDecrypter creates a new decrypter which uses the AWS KMS service to decrypt variables
func MakeKMSDecrypter() Decrypter {
	return &kmsDecrypter{
		kmsClient: kms.New(session.New(nil)),
	}
}

func (kd *kmsDecrypter) Decrypt(cipherText string) (string, error) {

	decodedBytes, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("Failed to encode cipher text to base64: %v", err)
	}

	params := &kms.DecryptInput{
		CiphertextBlob: decodedBytes,
	}

	response, err := kd.kmsClient.Decrypt(params)
	if err != nil {
		return "", fmt.Errorf("Failed to decrypt ciphertext with kms: %v", err)
	}
	// Plaintext is a byte array, so convert to string
	decrypted := string(response.Plaintext[:])

	return decrypted, nil
}
