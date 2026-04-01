locals {
  az_count         = length(data.aws_availability_zones.this.ids)
  subnets_count    = 4 * local.az_count
  subnets          = cidrsubnets(var.vpc_cidr, [for i in range(local.subnets_count) : 8]...)
  private_subnets  = chunklist(local.subnets, local.subnets_count / 2)[0]
  public_subnets   = chunklist(local.subnets, local.subnets_count / 2)[1]
  intra_subnets    = chunklist(local.subnets, local.subnets_count / 2)[2]
  database_subnets = chunklist(local.subnets, local.subnets_count / 2)[3]
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 6.6"
  name    = var.name
  cidr    = var.vpc_cidr

  private_subnets  = local.private_subnets
  public_subnets   = local.public_subnets
  intra_subnets    = local.intra_subnets
  database_subnets = local.database_subnets
  azs              = [for i in data.aws_availability_zones.this.zone_ids : "${i}"]

  enable_nat_gateway = true
  single_nat_gateway = true

  tags = var.tags
  private_subnet_tags = {
    SubnetType = "Private"
  }
  public_subnet_tags = {
    SubnetType = "Public"
  }
  database_subnet_tags = {
    SubnetType = "Database"
  }
  intra_subnet_tags = {
    SubnetType = "Intra"
  }
}

data "aws_region" "this" {}
data "aws_availability_zones" "this" {
  state = "available"
  filter {
    name   = "region-name"
    values = toset([data.aws_region.this.name])
  }
}
