resource "aws_security_group" "complex_sg" {
  name_prefix = "${local.service_name}-"
  description = "Complex security group for ${local.service_name}"
  vpc_id      = aws_vpc.main.id

  # Multiple ingress rules with different configurations
  ingress {
    description = "HTTP from anywhere"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description     = "HTTPS from anywhere"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    cidr_blocks     = ["0.0.0.0/0"]
    prefix_list_ids = ["pl-12345678"]
  }

  ingress {
    description              = "SSH from management"
    from_port                = 22
    to_port                  = 22
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.management.id
  }

  ingress {
    description = "Custom app port"
    from_port   = 8080
    to_port     = 8090
    protocol    = "tcp"
    cidr_blocks = [
      "10.0.0.0/8",
      "172.16.0.0/12",
      "192.168.0.0/16"
    ]
  }

  # Multiple egress rules
  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description              = "Database access"
    from_port                = 5432
    to_port                  = 5432
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.database.id
  }

  dynamic "ingress" {
    for_each = var.additional_ports
    content {
      description = "Dynamic port ${ingress.value}"
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = ["10.0.0.0/8"]
    }
  }

  tags = merge(local.common_tags, {
    Name = "${local.service_name}-security-group"
    Type = "application"
  })
}
resource "aws_security_group" "database" {
  name_prefix = "database-"
  vpc_id      = aws_vpc.main.id
  tags        = local.common_tags
}
resource "aws_security_group" "management" {
  name_prefix = "management-"
  vpc_id      = aws_vpc.main.id
  tags        = local.common_tags
}