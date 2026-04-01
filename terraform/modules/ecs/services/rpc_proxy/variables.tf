variable "name" {
  type        = string
  description = "Resources name prefix"
}

variable "tags" {
  type        = map(string)
  description = "Tags for resources"
  default     = {}
}

variable "subnet_ids" {
  type        = set(string)
  description = "Subnet IDs the service tasks will run in"
}

variable "cluster_id" {
  type        = string
  description = "ECS cluster ID to create the service in"
}

variable "image" {
  type        = string
  description = "Docker image URI for the rpc-proxy container (e.g. ECR URI with tag)"
}

variable "app_version" {
  type        = string
  default     = null
  description = "Application version label (informational, applied as a tag)"
}

variable "secret_arn" {
  type        = string
  description = "Secrets Manager secret ARN with application config values"
}

variable "domain" {
  type        = string
  description = "Domain at which the proxy should be reachable (used for ALB rule host-header)"
}

variable "ingress_rules" {
  type = list(object({
    from_port   = number
    to_port     = number
    ip_protocol = string
    cidr_ipv4   = optional(string, "0.0.0.0/0")
    description = optional(string, "")
  }))
  description = "Ingress security group rules for the service. Defaults to allowing container port from anywhere."
  default     = []
}
