terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~>6"
    }
  }
}

provider "aws" {}

locals {
  env  = ["main"]
  name = "twt-gzttkorg"
}

resource "aws_s3_bucket" "terraform_state" {
  for_each = local.env
  bucket   = "${local.name}-${each.value}-tfstate"

  lifecycle {
    prevent_destroy = false
  }

  tags = {
    Terraform = "1"
    Env       = each.value
  }
}

resource "aws_s3_bucket_versioning" "terraform_state" {
  for_each = aws_s3_bucket.terraform_state
  bucket   = each.value.id

  versioning_configuration {
    status = "Enabled"
  }
}
