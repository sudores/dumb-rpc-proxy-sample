module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "~>5.1"

  domain_name = var.domain

  subject_alternative_names = [
    "*.${var.domain}"
  ]

  create_route53_records = false
  wait_for_validation    = false
  validation_method      = "DNS"

  tags = merge({
    Name      = "${var.domain}"
    Terraform = "1"
  }, var.tags)
}
