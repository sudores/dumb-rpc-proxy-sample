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
  description = "Docker image URI for the rpc-proxy container"
}

variable "app_version" {
  type        = string
  default     = null
  description = "Application version label applied as a tag"
}

variable "desired_count" {
  type        = number
  default     = 1
  description = "Number of task instances to run"
}

variable "upstream_url" {
  type        = string
  default     = "https://polygon.drpc.org"
  description = "Upstream RPC URL passed as UPSTREAM_URL env var to the container"
}

variable "domain" {
  type        = string
  description = "Domain for the ALB host-header rule"
}

variable "ingress_rules" {
  type = list(object({
    from_port   = number
    to_port     = number
    ip_protocol = string
    cidr_ipv4   = optional(string, "0.0.0.0/0")
    description = optional(string, "")
  }))
  description = "Ingress security group rules for the service"
  default     = []
}
