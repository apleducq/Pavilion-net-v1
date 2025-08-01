# Infrastructure as Code

This directory contains Terraform configurations for the B2B Trust Broker platform infrastructure.

## Structure

- **environments/** - Environment-specific configurations
  - **dev/** - Development environment
  - **staging/** - Staging environment  
  - **production/** - Production environment
- **modules/** - Reusable Terraform modules
  - **vpc/** - VPC and networking
  - **eks/** - Kubernetes clusters
  - **rds/** - Database instances
  - **redis/** - Redis clusters
  - **alb/** - Load balancers
- **scripts/** - Infrastructure automation scripts

## Prerequisites

- Terraform >= 1.0
- AWS CLI configured
- kubectl (for Kubernetes operations)

## Quick Start

```bash
# Initialize terraform
terraform init

# Plan changes
terraform plan

# Apply changes
terraform apply
```

## Environment Management

Each environment has its own configuration:
- `environments/dev/` - Development infrastructure
- `environments/staging/` - Staging infrastructure  
- `environments/production/` - Production infrastructure

## Security

- All sensitive values are stored in AWS Secrets Manager
- IAM roles follow least privilege principle
- Network security groups are configured for minimal access 