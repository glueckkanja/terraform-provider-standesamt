# Azure Certificate Authentication

## Supported Certificate Formats

The Standesamt Terraform provider supports Azure Service Principal authentication using client certificates. For security reasons, only modern encryption formats are supported.

### Recommended Format: PKCS#12 (Preferred)

**PKCS#12** (`.pfx` or `.p12`) is the **recommended format** for Azure Service Principal certificates:

✅ **Advantages:**
- Single file contains both certificate and private key
- Native Azure Portal format (downloaded as `.pfx`)
- Built-in password encryption (3DES or AES)
- Supports full certificate chains (intermediate CAs)
- Industry standard for certificate distribution
- Fully supported by Azure SDK
- Uses modern go-pkcs12 library (software.sslmate.com/src/go-pkcs12)

```hcl
provider "standesamt" {
  location_source = "azure"
  
  azure = {
    subscription_id             = "..."
    tenant_id                   = "..."
    client_id                   = "..."
    client_certificate_path     = "/path/to/certificate.pfx"
    client_certificate_password = var.cert_password
  }
}
```

### Alternative Format: PEM (PKCS#8)

If you prefer PEM format, the following formats are supported:

1. **Combined PEM file** (certificate + private key in one file)
   - Certificate: `-----BEGIN CERTIFICATE-----`
   - Private Key (unencrypted): `-----BEGIN PRIVATE KEY-----` (PKCS#8)
   - Private Key (unencrypted): `-----BEGIN RSA PRIVATE KEY-----` (PKCS#1)
   - Private Key (unencrypted): `-----BEGIN EC PRIVATE KEY-----` (EC)
   - Private Key (encrypted): `-----BEGIN ENCRYPTED PRIVATE KEY-----` (PKCS#8 with AES)

2. **Separate PEM files** are NOT supported - combine them first

### Unsupported Formats

❌ **Legacy PEM encryption is NOT supported** for security reasons:
- Old-style encrypted keys with `DEK-Info` headers
- Keys using DES, 3DES, or other weak encryption algorithms

## Converting Between Formats

### Convert PEM to PKCS#12 (Recommended)

```bash
# Combine certificate and private key into PKCS#12
openssl pkcs12 -export \
  -in certificate.pem \
  -inkey private_key.pem \
  -out certificate.pfx \
  -passout pass:YourPassword

### Convert PKCS#12 to PEM

```bash
# Extract certificate and private key from PKCS#12
openssl pkcs12 -in certificate.pfx \
  -out combined.pem \
  -nodes  # Use -nodes for unencrypted output, omit for encrypted
```

### Convert Legacy PEM to PKCS#12

```bash
# If you have an old encrypted PEM key with DEK-Info
# First convert to modern PKCS#8, then to PKCS#12
openssl pkcs8 -topk8 -v2 aes-256-cbc \
  -in old_key.pem \
  -out new_key.pem

openssl pkcs12 -export \
  -in certificate.pem \
  -inkey new_key.pem \
  -out certificate.pfx
```

## Generating New Certificates

### Option 1: Generate PKCS#12 directly (Recommended)

```bash
# Generate private key and self-signed certificate in one step
openssl req -x509 -newkey rsa:4096 \
  -keyout temp_key.pem \
  -out temp_cert.pem \
  -days 365 \
  -subj "/CN=MyServicePrincipal" \
  -passout pass:temppass

# Convert to PKCS#12
openssl pkcs12 -export \
  -in temp_cert.pem \
  -inkey temp_key.pem \
  -passin pass:temppass \
  -out certificate.pfx \
  -passout pass:YourSecurePassword

# Clean up temporary files
rm temp_key.pem temp_cert.pem
```

### Option 2: Generate PEM format

```bash
# Generate private key and certificate
openssl req -x509 -newkey rsa:4096 \
  -keyout private_key.pem \
  -out certificate.pem \
  -days 365 \
  -subj "/CN=MyServicePrincipal"

# Combine into single file
cat certificate.pem private_key.pem > combined.pem
```

## Registering Certificate with Azure

After generating your certificate, register it with Azure AD:

```bash
# Extract public certificate from PKCS#12
openssl pkcs12 -in certificate.pfx \
  -clcerts -nokeys \
  -out public_cert.pem

# Register with Azure CLI
az ad sp credential reset \
  --id <client-id> \
  --cert @public_cert.pem
```

## Security Best Practices

1. **Use encrypted certificates** when stored on disk
2. **Use unencrypted certificates** only in secure environments (e.g., CI/CD with secrets management)
3. **Never commit** private keys to version control
4. **Use Key Vault** or similar secret management solutions in production
5. **Rotate certificates** regularly (recommended: every 90 days)
6. **Use strong passwords** when encrypting private keys (minimum 16 characters)

## Troubleshooting

### Error: "legacy encrypted private key format detected"

Your certificate uses old DES/3DES encryption. Convert it using the command above.

### Error: "encrypted private key found but no password provided"

Your private key is encrypted but you didn't provide `client_certificate_password`. Either:
- Provide the password in the provider configuration
- Convert to an unencrypted key (for secure automated deployments)

### Error: "failed to parse PKCS8 encrypted private key"

Your certificate might not be in PKCS#8 format or uses unsupported encryption. Convert it using:
```bash
openssl pkcs8 -topk8 -v2 aes-256-cbc -in old.pem -out new.pem
```

