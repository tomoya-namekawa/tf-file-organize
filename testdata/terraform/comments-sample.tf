# Main configuration file for the project
# This file contains the core infrastructure setup

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

# AWS Provider configuration
# Using us-west-2 region for all resources
provider "aws" {
  region = "us-west-2"
}

# Instance type variable
# Allows customization of EC2 instance size
variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

# Common local values
# Shared across all resources
locals {
  common_tags = {
    Environment = "dev"
    Project     = "tf-file-organize"
  }
}

# Data source for Ubuntu AMI
# Gets the latest Ubuntu 20.04 AMI
data "aws_ami" "ubuntu_simple" {
  most_recent = true
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"]
  }
  owners = ["099720109477"]
}

# Security group for web traffic
resource "aws_security_group" "simple" {
  name_prefix = "simple-"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

# Main EC2 instance
# This is the primary server
resource "aws_instance" "simple" {
  ami           = data.aws_ami.ubuntu_simple.id
  instance_type = var.instance_type

  vpc_security_group_ids = [aws_security_group.simple.id]

  tags = merge(local.common_tags, {
    Name = "simple-server"
  })
}

# Output values
# These values are exported after apply
output "simple_instance_id" {
  description = "ID of the simple EC2 instance"
  value       = aws_instance.simple.id
}