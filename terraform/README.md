# Overview
This project provides a modular and scalable Infrastructure as Code (IaC) setup
using Terraform.It supports multiple environments (dev, stage, production) using
Terraform workspaces, and organizes components into reusable modules

>📈 An infrastructure diagram illustrating the relationships between
>key components (VPC, ECS, ALB, RDS, etc.)

![Infrastructure Overview](images/Infra.jpg)

## Key Conepts

### Workspaces & Deployment
We use Terraform workspaces to manage isolated environments:
- dev
- stage
- prod

Each workspace maintains its own state and allows deploying infrastructure 
independently for different stages of the software lifecycle.

To initialize, select a workspace and apply infrastructure:
``` bash
terraform init
terraform workspace list
terraform workspace select dev
terraform plan
terraform apply
```
(for additional info write `terraform workspace` to get a list of commands)

## Project Structure
```
.
├── main.tf               # Root module to orchestrate everything
├── provider.tf           # Provider configuration (AWS, etc.)
├── s3_state/             # Remote state configuration using S3 backend
├── modules/              # Reusable infrastructure modules
│   ├── vpc/              # Virtual Private Cloud setup
│   ├── rds/              # Relational Database Service (MySQL)
│   ├── ecs/              # ECS Cluster and services
│   │   └── services/     # Individual ECS services: backend, alertmanager, monitoring, etc.
│   ├── alb/              # Application Load Balancer
│   ├── cloudfront/       # CDN configuration
│   ├── acm/              # SSL certificates using AWS Certificate Manager
│   ├── elasticache/      # Redis configuration
│   ├── s3/               # S3 buckets for frontend
│   └── s3_back/          # S3 bucket for backend-specific needs
```

## Remote State Management
State is stored remotely using S3 to enable team collaboration and
consistency across environments. See the **s3_state/** directory for additional info

## Module Overview

Each folder in modules/ is a self-contained Terraform module with:
- main.tf: Resource definitions
- variables.tf: Input variables
- outputs.tf: Output values

### Modules:

**modules/vpc**
- Sets up a private and public VPC with subnets using `terraform-aws-modules/vpc/aws`

**modules/ecs**

Provisions and ECS clsuter and deploys services like:
- backend
- socketi
- alertmanager
- monitroing (Prometheus, exporters, etc.)

**modules/alb**
- Configures Application Load Balancer, target groups and listeners for routing traffic

**modules/rds**
- Creates an RDS MySQL instance with appropriate networking and credentials

**modules/elasticache**
- Deploys an Amazon ElastiCache for Redis cluster

**modules/acm**
- Requests and validates TLS/SSL certificates used for secure communication

**modules/cloudfront**
- Sets up CloudFront distribution for global content delivery and caching
