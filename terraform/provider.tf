terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~>6.36"
    }
  }
  required_version = "~>1.11"
  backend "s3" {
    bucket = "twt-main-gzttkorg"
    key    = "tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"
  default_tags {
    tags = {
      Env       = local.env,
      Terraform = true,
    }
  }
}
