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
	"software.sslmate.com/src/go-pkcs12"
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

// parseCertificate parses a certificate file and returns the certificates and private key.
//
// Supported formats:
//   - PKCS#12/PFX (.pfx, .p12): Recommended format for Azure certificates (auto-detected)
//     Uses software.sslmate.com/src/go-pkcs12 for full PKCS#12 support including certificate chains
//   - PEM format: Combined certificate and private key in PEM encoding
//
// PEM Private Key formats:
//   - PRIVATE KEY (PKCS8), RSA PRIVATE KEY (PKCS1), EC PRIVATE KEY blocks
//   - ENCRYPTED PRIVATE KEY (PKCS8 with password)
//
// Security Note: For PEM format, only modern PKCS8 encrypted private keys are supported.
// Legacy PEM encryption (using weak DES/3DES) is not supported for security reasons.
// For PKCS#12, modern encryption algorithms (AES) are supported via go-pkcs12.
//
// Recommended: Use PKCS#12 format for best compatibility with Azure:
//
//	openssl pkcs12 -export -in certificate.pem -inkey private_key.pem -out certificate.pfx
func parseCertificate(data []byte, password string) ([]*x509.Certificate, interface{}, error) {
	// Try PKCS#12 format first (common for Azure certificates)
	if isPKCS12(data) {
		return parsePKCS12(data, password)
	}

	// Fall back to PEM format
	return parsePEM(data, password)
}

// isPKCS12 checks if the data is in PKCS#12 format
// PKCS#12 files typically start with a specific ASN.1 sequence
func isPKCS12(data []byte) bool {
	// PKCS#12 files start with 0x30 (ASN.1 SEQUENCE)
	// and are binary (not PEM text)
	if len(data) < 4 {
		return false
	}
	// Check if it looks like binary ASN.1 (not PEM text)
	// PEM files start with "-----BEGIN"
	if strings.HasPrefix(string(data), "-----BEGIN") {
		return false
	}
	// PKCS#12 starts with 0x30 0x82 or 0x30 0x83 (ASN.1 SEQUENCE with length)
	return data[0] == 0x30 && (data[1] == 0x82 || data[1] == 0x83 || data[1] == 0x84)
}

// parsePKCS12 parses a PKCS#12 formatted certificate.
// Uses software.sslmate.com/src/go-pkcs12 which is the modern, maintained alternative
// to the frozen golang.org/x/crypto/pkcs12 package.
func parsePKCS12(data []byte, password string) ([]*x509.Certificate, interface{}, error) {
	// DecodeChain supports full certificate chains and modern encryption algorithms
	key, cert, caCerts, err := pkcs12.DecodeChain(data, password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode PKCS#12 certificate (ensure password is correct): %w", err)
	}

	if cert == nil {
		return nil, nil, fmt.Errorf("no certificate found in PKCS#12 data")
	}
	if key == nil {
		return nil, nil, fmt.Errorf("no private key found in PKCS#12 data")
	}

	// Combine leaf certificate with CA chain
	allCerts := []*x509.Certificate{cert}
	if len(caCerts) > 0 {
		allCerts = append(allCerts, caCerts...)
	}

	return allCerts, key, nil
}

// parsePEM parses a PEM-encoded certificate and returns the certificates and private key.
func parsePEM(data []byte, password string) ([]*x509.Certificate, interface{}, error) {
	var certs []*x509.Certificate
	var key interface{}
	hasPassword := password != ""

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

		case block.Type == "ENCRYPTED PRIVATE KEY":
			// PKCS8 encrypted private key (modern encryption: AES, etc.)
			if !hasPassword {
				return nil, nil, fmt.Errorf("encrypted private key found but no password provided")
			}

			// Try to parse with pkcs8.ParsePKCS8PrivateKey which handles encrypted PKCS8
			parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse PKCS8 encrypted private key: %w (ensure the certificate is in PKCS8 format with modern encryption)", err)
			}
			key = parsedKey

		case strings.Contains(block.Type, "PRIVATE KEY"):
			var err error

			// Check if this is actually an old-style encrypted block
			// These have headers like "DEK-Info: DES-EDE3-CBC,..." which we don't support
			if block.Headers != nil {
				if _, hasEncryption := block.Headers["DEK-Info"]; hasEncryption {
					return nil, nil, fmt.Errorf("legacy encrypted private key format detected (weak DES/3DES encryption). Please convert to PKCS8 format: openssl pkcs8 -topk8 -v2 aes-256-cbc -in old_key.pem -out new_key.pem")
				}
			}

			// Try different unencrypted key formats
			// PKCS8 format (recommended)
			if key, err = x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
				continue
			}
			// PKCS1 RSA format
			if key, err = x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
				continue
			}
			// EC private key format
			if key, err = x509.ParseECPrivateKey(block.Bytes); err == nil {
				continue
			}
			return nil, nil, fmt.Errorf("failed to parse private key: unsupported format or corrupted data")
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
