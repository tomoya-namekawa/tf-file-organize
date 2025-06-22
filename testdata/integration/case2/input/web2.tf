resource "aws_instance" "web2" {
  ami           = "ami-12345"
  instance_type = "t3.small"

  tags = {
    Name = "web2"
  }
}

output "web2_id" {
  value = aws_instance.web2.id
}