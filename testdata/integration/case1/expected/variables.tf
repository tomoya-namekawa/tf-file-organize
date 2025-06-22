variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}
variable "key_name" {
  description = "EC2 Key Pair name"
  type        = string
}