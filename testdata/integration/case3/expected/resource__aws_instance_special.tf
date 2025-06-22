resource "aws_instance_special" "admin" {
  ami           = "ami-67890"
  instance_type = "t3.small"

  tags = merge(local.common_tags, {
    Name = "admin-server"
  })
}