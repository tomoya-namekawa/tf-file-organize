resource "aws_security_group" "complex_sg" {
  description = "Complex security group for ${local.service_name}"
  name_prefix = "${local.service_name}-"
  tags        = ""
  vpc_id      = aws_vpc.main.id
  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP from anywhere"
    from_port   = 80
    protocol    = "tcp"
    to_port     = 80
  }
  ingress {
    cidr_blocks     = ["0.0.0.0/0"]
    description     = "HTTPS from anywhere"
    from_port       = 443
    prefix_list_ids = ["pl-12345678"]
    protocol        = "tcp"
    to_port         = 443
  }
  ingress {
    description              = "SSH from management"
    from_port                = 22
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.management.id
    to_port                  = 22
  }
  ingress {
    cidr_blocks = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
    description = "Custom app port"
    from_port   = 8080
    protocol    = "tcp"
    to_port     = 8090
  }
  egress {
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
    from_port   = 0
    protocol    = "-1"
    to_port     = 0
  }
  egress {
    description              = "Database access"
    from_port                = 5432
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.database.id
    to_port                  = 5432
  }
  dynamic "ingress" {
    for_each = var.additional_ports
  }
}

resource "aws_security_group" "database" {
  name_prefix = "database-"
  tags        = local.common_tags
  vpc_id      = aws_vpc.main.id
}

resource "aws_security_group" "management" {
  name_prefix = "management-"
  tags        = local.common_tags
  vpc_id      = aws_vpc.main.id
}
