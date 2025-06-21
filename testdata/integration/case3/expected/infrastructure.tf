resource "aws_security_group" "web" {
  name_prefix = "web-"
  tags        = local.common_tags
  vpc_id      = aws_vpc.main.id
  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    from_port   = 80
    protocol    = "tcp"
    to_port     = 80
  }
}

resource "aws_subnet" "public" {
  cidr_block = "10.0.1.0/24"
  tags       = local.common_tags
  vpc_id     = aws_vpc.main.id
}

resource "aws_vpc" "main" {
  cidr_block = var.vpc_cidr
  tags       = local.common_tags
}
