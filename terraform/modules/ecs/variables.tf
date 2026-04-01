variable "name" {
  type        = string
  description = "Resources name"
}

variable "tags" {
  type        = map(string)
  description = "Tags for resources"
  default     = null
}

variable "subnet_ids" {
  type        = list(string)
  description = "Subnet IDs for the EC2 autoscaling group"
}

variable "instance_type" {
  type        = string
  default     = "t3.small"
  description = "EC2 instance type for ECS container instances"
}
