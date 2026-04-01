variable "domain" {
  type        = string
  description = "domain for cert"
}

variable "tags" {
  type        = map(any)
  description = "Resources tags"
  default     = {}
}
