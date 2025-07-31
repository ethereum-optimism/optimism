# Alchemy Pay API Deployment Guide

This guide provides step-by-step instructions for deploying the Alchemy Pay API using Serverless Framework with custom domain configuration.

## Prerequisites

1. AWS Account with appropriate permissions
2. AWS CLI installed and configured
3. Node.js (v14 or later)
4. Python 3.9
5. Serverless Framework CLI
6. Domain name registered in Route53 (or ability to create one)

## Initial Setup

1. Install dependencies:
   ```bash
   # Install serverless framework globally
   npm install -g serverless

   # Install project dependencies
   pnpm install

   # Install Python dependencies
   pip install -r requirements.txt
   ```

2. Configure environment:
   ```bash
   # Copy example environment files
   cp env-dev.example.yml env-dev.yml
   cp env-mainnet.example.yml env-mainnet.yml
   ```

3. Update environment files with your values:
   - `env-dev.yml`:
     ```yaml
     APP_ID: "your-app-id"
     SECRET_KEY: "your-secret-key"
     BASE_URL: "https://ramptest.alchemypay.org/"
     STAGE: "dev"
     CORS_ORIGINS: "https://staging.hub.boba.network,https://hub-dev.boba.network,http://localhost:3000"
     DOMAIN_NAME: "alchemypay-api.sepolia-dev.boba.network"
     CERTIFICATE_NAME: "*.sepolia-dev.boba.network"
     ```
   - `env-mainnet.yml`:
     ```yaml
     APP_ID: "your-mainnet-app-id"
     SECRET_KEY: "${env:MAINNET_SECRET_KEY}"
     BASE_URL: "https://ramp.alchemypay.org/"
     STAGE: "mainnet"
     CORS_ORIGINS: "https://hub.boba.network,https://app.boba.network"
     DOMAIN_NAME: "alchemypay-api.boba.network"
     CERTIFICATE_NAME: "*.boba.network"
     ```

## SSL Certificate Setup

1. Create SSL certificate in AWS Certificate Manager (ACM):
   ```bash
   # Navigate to AWS Certificate Manager in the AWS Console
   # Request a certificate
   # Add your domain (e.g., *.boba.network for wildcard)
   # Choose DNS validation
   # Add tags if needed
   # Request the certificate
   ```

2. Validate the certificate:
   - Follow the DNS validation steps provided by ACM
   - Add the CNAME records to your Route53 hosted zone
   - Wait for validation to complete (can take up to 30 minutes)

## Route53 Setup

1. Create a hosted zone (if not exists):
   ```bash
   # In AWS Console:
   # Go to Route53 → Hosted zones → Create hosted zone
   # Enter your domain name (e.g., boba.network)
   # Choose "Public hosted zone"
   # Create
   ```

2. Note the nameservers and update your domain registrar if needed.

## Deployment

1. Deploy to development:
   ```bash
   serverless deploy --stage dev --region us-east-1
   ```

2. Deploy to mainnet:
   ```bash
   serverless deploy --stage mainnet --region us-east-1
   ```

The deployment process will:
1. Package your code
2. Upload to S3
3. Create/update CloudFormation stack
4. Create/update API Gateway
5. Create/update Lambda function
6. Set up custom domain mapping

## Custom Domain Setup

The serverless-domain-manager plugin will automatically:
1. Create API Gateway custom domain
2. Create Route53 record
3. Map the API to the custom domain

To create the custom domain manually:
```bash
serverless create_domain --stage dev
# or
serverless create_domain --stage mainnet
```

## Verification

1. Check deployment status:
   ```bash
   serverless info --stage dev
   # or
   serverless info --stage mainnet
   ```

2. Test the endpoint:
   ```bash
   curl -X POST https://your-custom-domain/generate_alchemypay_url
   ```

## Troubleshooting

1. Certificate issues:
   - Ensure certificate is validated
   - Check certificate region matches API Gateway region
   - Verify certificate name in env file matches ACM

2. DNS issues:
   - Allow up to 48 hours for DNS propagation
   - Verify Route53 records are created
   - Check nameserver configuration

3. API Gateway issues:
   - Check CloudWatch logs
   - Verify CORS settings
   - Check API Gateway stage deployment

## Cleanup

To remove the deployment:
```bash
# Remove API deployment
serverless remove --stage dev

# Remove custom domain
serverless delete_domain --stage dev
```

## Security Considerations

1. Never commit environment files with real credentials
2. Use AWS KMS for sensitive values
3. Implement proper CORS settings
4. Use appropriate IAM roles
5. Regularly rotate credentials

## Monitoring

1. Set up CloudWatch alarms for:
   - Lambda errors
   - API Gateway 4xx/5xx errors
   - Lambda duration
   - Lambda concurrent executions

2. Enable X-Ray tracing if needed:
   ```yaml
   # In serverless.yml
   provider:
     tracing:
       apiGateway: true
       lambda: true
   ```

## Cost Optimization

1. Set appropriate Lambda timeout
2. Configure Lambda memory based on usage
3. Use provisioned concurrency if needed
4. Monitor API Gateway usage
5. Clean up unused resources

## Additional Resources

- [Serverless Framework Documentation](https://www.serverless.com/framework/docs/)
- [AWS Lambda Documentation](https://docs.aws.amazon.com/lambda/)
- [API Gateway Documentation](https://docs.aws.amazon.com/apigateway/)
- [Route53 Documentation](https://docs.aws.amazon.com/route53/)
