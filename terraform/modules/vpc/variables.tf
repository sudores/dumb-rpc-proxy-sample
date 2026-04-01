variable "name" {
  type        = string
  description = "Resources name"
}

variable "tags" {
  type        = map(string)
  description = "Tags for resoruces"
  default     = null
}

variable "vpc_cidr" {
  type        = string
  description = "VPC cidr to use. Minimim mask is 16"
}
