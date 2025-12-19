// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// CloudEnvironment represents the Azure cloud environment
type CloudEnvironment string

const (
	CloudEnvironmentPublic       CloudEnvironment = "public"
	CloudEnvironmentUSGovernment CloudEnvironment = "usgovernment"
	CloudEnvironmentChina        CloudEnvironment = "china"
)

// Config holds the Azure authentication configuration
type Config struct {
	// Authentication methods
	UseCli  bool
	UseMsi  bool
	UseOidc bool

	// Service Principal credentials
	ClientId                  string
	ClientSecret              string
	ClientCertificatePath     string
	ClientCertificatePassword string

	// Tenant and Subscription
	TenantId       string
	SubscriptionId string

	// Environment
	Environment CloudEnvironment
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.SubscriptionId == "" {
		return fmt.Errorf("subscription_id is required for Azure location source")
	}

	// Check if at least one auth method is configured or available
	hasServicePrincipal := c.ClientId != "" && (c.ClientSecret != "" || c.ClientCertificatePath != "")
	hasAuthMethod := c.UseCli || c.UseMsi || c.UseOidc || hasServicePrincipal

	if !hasAuthMethod {
		// Default to CLI auth if nothing is specified
		c.UseCli = true
	}

	return nil
}

// GetCloudConfig returns the Azure cloud configuration based on the environment
func (c *Config) GetCloudConfig() cloud.Configuration {
	switch c.Environment {
	case CloudEnvironmentUSGovernment:
		return cloud.AzureGovernment
	case CloudEnvironmentChina:
		return cloud.AzureChina
	default:
		return cloud.AzurePublic
	}
}

// GetCredential creates an Azure credential based on the configuration
func (c *Config) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	cloudConfig := c.GetCloudConfig()
	clientOpts := &azcore.ClientOptions{
		Cloud: cloudConfig,
	}

	// Try authentication methods in order of preference
	var credentials []azcore.TokenCredential

	// 1. Service Principal with Client Secret
	if c.ClientId != "" && c.ClientSecret != "" && c.TenantId != "" {
		cred, err := azidentity.NewClientSecretCredential(
			c.TenantId,
			c.ClientId,
			c.ClientSecret,
			&azidentity.ClientSecretCredentialOptions{
				ClientOptions: *clientOpts,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}
		return cred, nil
	}

	// 2. Service Principal with Client Certificate
	if c.ClientId != "" && c.ClientCertificatePath != "" && c.TenantId != "" {
		certData, err := os.ReadFile(c.ClientCertificatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read client certificate: %w", err)
		}

		certs, key, err := parseCertificate(certData, c.ClientCertificatePassword)
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate: %w", err)
		}

		cred, err := azidentity.NewClientCertificateCredential(
			c.TenantId,
			c.ClientId,
			certs,
			key,
			&azidentity.ClientCertificateCredentialOptions{
				ClientOptions: *clientOpts,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create client certificate credential: %w", err)
		}
		return cred, nil
	}

	// 3. OIDC (Workload Identity)
	if c.UseOidc {
		cred, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
			ClientOptions: *clientOpts,
			ClientID:      c.ClientId,
			TenantID:      c.TenantId,
		})
		if err == nil {
			credentials = append(credentials, cred)
		}
	}

	// 4. Managed Identity
	if c.UseMsi {
		opts := &azidentity.ManagedIdentityCredentialOptions{
			ClientOptions: *clientOpts,
		}
		if c.ClientId != "" {
			opts.ID = azidentity.ClientID(c.ClientId)
		}
		cred, err := azidentity.NewManagedIdentityCredential(opts)
		if err == nil {
			credentials = append(credentials, cred)
		}
	}

	// 5. Azure CLI
	if c.UseCli {
		cred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{
			TenantID: c.TenantId,
		})
		if err == nil {
			credentials = append(credentials, cred)
		}
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("no valid Azure authentication method configured")
	}

	if len(credentials) == 1 {
		return credentials[0], nil
	}

	// Use ChainedTokenCredential if multiple methods are available
	chain, err := azidentity.NewChainedTokenCredential(credentials, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chained credential: %w", err)
	}

	return chain, nil
}

// parseCertificate parses a PEM-encoded certificate and returns the certificates and private key
func parseCertificate(data []byte, password string) ([]*x509.Certificate, interface{}, error) {
	var certs []*x509.Certificate
	var key interface{}

	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		data = rest

		switch {
		case strings.Contains(block.Type, "CERTIFICATE"):
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
			}
			certs = append(certs, cert)

		case strings.Contains(block.Type, "PRIVATE KEY"):
			var err error
			blockBytes := block.Bytes

			// Handle encrypted private key
			if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck
				blockBytes, err = x509.DecryptPEMBlock(block, []byte(password)) //nolint:staticcheck
				if err != nil {
					return nil, nil, fmt.Errorf("failed to decrypt private key: %w", err)
				}
			}

			// Try different key formats
			if key, err = x509.ParsePKCS8PrivateKey(blockBytes); err == nil {
				continue
			}
			if key, err = x509.ParsePKCS1PrivateKey(blockBytes); err == nil {
				continue
			}
			if key, err = x509.ParseECPrivateKey(blockBytes); err == nil {
				continue
			}
			return nil, nil, fmt.Errorf("failed to parse private key")
		}
	}

	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificates found in PEM data")
	}
	if key == nil {
		return nil, nil, fmt.Errorf("no private key found in PEM data")
	}

	return certs, key, nil
}
