resource "aws_subnet" "public" {
  count             = 2
  vpc_id            = "vpc-0a1b2c3d"
  cidr_block        = "10.0.${count.index + 10}.0/24"
  availability_zone = ["us-east-1a", "us-east-1b"][count.index]
  tags              = { Name = "public-${count.index}", Env = "prod" }
}

resource "aws_security_group" "web" {
  name        = "web"
  description = "Managed by Terraform"
  vpc_id      = "vpc-0a1b2c3d"
  tags        = { Env = "prod", Owner = "infra" }
}

output "subnet_ids" {
  value = aws_subnet.public[*].id
}

output "security_group_id" {
  value = aws_security_group.web.id
}
