variable "name" {
  type        = string
  description = "Resources name"
}

variable "tags" {
  type        = map(string)
  description = "Tags for resources"
  default     = null
}

variable "vpc_id" {
  type        = string
  description = "VPC ID the ALB will reside in"
}

variable "subnet_ids" {
  type        = set(string)
  description = "Subnet IDs the ALB will reside in"
}

variable "cert_arn" {
  type        = string
  description = "Certificate ARN the ALB will use for HTTPS"
}

variable "rules" {
  type        = map(any)
  description = "ALB HTTPS listener rules, compatible with terraform-aws-modules/alb rule format"
  default     = {}
}
