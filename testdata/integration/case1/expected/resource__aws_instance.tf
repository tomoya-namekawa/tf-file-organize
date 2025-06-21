resource "aws_instance" "web" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  key_name               = var.key_name
  tags                   = ""
  vpc_security_group_ids = [aws_security_group.web.id]
}
