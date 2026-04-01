locals {
  prefix      = "twt-gzttkorg"
  env         = terraform.workspace
  name        = "${local.prefix}-${local.env}"
  root_domain = "gzttk.org"
  domain = {
    dev  = "twt-polygon-rpc.${terraform.workspace}.${local.root_domain}"
    main = "twt-polygon-rpc.${local.root_domain}"
  }
}

module "vpc" {
  source   = "./modules/vpc"
  name     = local.name
  vpc_cidr = "10.0.0.0/16"
}

module "acm_cert" {
  source = "./modules/acm/"
  domain = local.domain[local.env]
}

module "alb" {
  source     = "./modules/alb"
  name       = local.name
  cert_arn   = module.acm_cert.cert_arn
  subnet_ids = module.vpc.public_subnet_ids
  vpc_id     = module.vpc.vpc_id
  rules      = module.rpc_proxy.lb_rules
}

module "ecs" {
  source     = "./modules/ecs"
  name       = local.name
  subnet_ids = module.vpc.private_subnet_ids
}

module "rpc_proxy" {
  source     = "./modules/ecs/services/rpc_proxy"
  name       = local.name
  subnet_ids = module.vpc.private_subnet_ids
  cluster_id = module.ecs.cluster_id
  domain     = local.domain[local.env]
  image      = "vepl/twt-rpc-proxy:latest" # replace with ECR image URI
  ingress_rules = [
    {
      from_port   = 8080
      to_port     = 8080
      ip_protocol = "tcp"
      cidr_ipv4   = module.vpc.vpc_cidr_block
      description = "Allow ALB to reach rpc-proxy"
    },
  ]
}
