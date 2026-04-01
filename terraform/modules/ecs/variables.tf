variable "name" {
  type        = string
  description = "Resources name"
}

variable "tags" {
  type        = map(string)
  description = "Tags for resoruces"
  default     = null
}

variable "subnet_ids" {
  type        = set(string)
  description = "Set of subnets cluster will rezide in"
}
