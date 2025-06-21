resource "aws_instance" "web" {
  ami                    = "ami-12345"
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.public.id
  tags                   = ""
  vpc_security_group_ids = [aws_security_group.web.id]
}
