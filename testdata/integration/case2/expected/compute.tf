resource "aws_instance" "web1" {
  ami           = "ami-12345"
  instance_type = "t3.micro"
  tags = {
    Name = "web1"
  }
}

resource "aws_instance" "web2" {
  ami           = "ami-12345"
  instance_type = "t3.small"
  tags = {
    Name = "web2"
  }
}
