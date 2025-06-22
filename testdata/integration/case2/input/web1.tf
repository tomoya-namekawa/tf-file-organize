terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-west-2"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "project_name" {
  description = "Project name"
  type        = string
}

resource "aws_instance" "web1" {
  ami           = "ami-12345"
  instance_type = "t3.micro"

  tags = {
    Name = "web1"
  }
}

output "web1_id" {
  value = aws_instance.web1.id
}