locals {
  supported_subnet_ids = toset([
    for id, s in data.aws_subnet.all : id
    if contains(toset(data.aws_ec2_instance_type_offerings.this.locations), s.availability_zone)
  ])

  user_data = <<-EOT
    #!/bin/bash
    cat <<EOF >> /etc/ecs/ecs.config
    ECS_CLUSTER=${var.name}
    EOF
  EOT
}

module "ecs" {
  source  = "terraform-aws-modules/ecs/aws"
  version = "~>7.5"

  cluster_name = var.name

  capacity_providers = {
    "${terraform.workspace}-EC2" = {
      auto_scaling_group_provider = {
        auto_scaling_group_arn         = module.autoscaling.autoscaling_group_arn
        managed_termination_protection = "ENABLED"
        managed_scaling = {
          maximum_scaling_step_size = 2
          minimum_scaling_step_size = 1
          status                    = "ENABLED"
          target_capacity           = 70
        }
      }
    }
  }

  default_capacity_provider_strategy = {
    "${terraform.workspace}-EC2" = {
      weight = 60
    }
  }

  tags = var.tags
}

module "autoscaling" {
  source  = "terraform-aws-modules/autoscaling/aws"
  version = "~>9.0"

  name                            = var.name
  image_id                        = jsondecode(data.aws_ssm_parameter.ecs_optimized_ami.value)["image_id"]
  ignore_desired_capacity_changes = true

  vpc_zone_identifier = local.supported_subnet_ids
  health_check_type   = "EC2"
  min_size            = 1
  max_size            = 4
  desired_capacity    = 2
  user_data           = base64encode(local.user_data)

  instance_type   = var.instance_type
  security_groups = [aws_security_group.this.id]

  create_iam_instance_profile = true
  iam_role_name               = "${var.name}-asg"
  iam_role_description        = "ECS role for ${var.name}"
  iam_role_policies = {
    AmazonEC2ContainerServiceforEC2Role = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
    AmazonSSMManagedInstanceCore        = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
    CloudWatchLogsFullAccess            = "arn:aws:iam::aws:policy/CloudWatchLogsFullAccess"
  }
  protect_from_scale_in = true

  tags                   = var.tags
  autoscaling_group_tags = var.tags
}

resource "aws_security_group" "this" {
  name        = "asg-${var.name}"
  description = "Security group for ECS autoscaling group"
  vpc_id      = data.aws_vpc.this.id
  tags        = var.tags
}

resource "aws_vpc_security_group_ingress_rule" "this" {
  security_group_id = aws_security_group.this.id
  cidr_ipv4         = data.aws_vpc.this.cidr_block
  from_port         = 0
  to_port           = 65535
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "this" {
  security_group_id = aws_security_group.this.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

data "aws_ec2_instance_type_offerings" "this" {
  filter {
    name   = "instance-type"
    values = [var.instance_type]
  }
  location_type = "availability-zone"
}

data "aws_subnet" "all" {
  for_each = var.subnet_ids
  id       = each.value
}

data "aws_ssm_parameter" "ecs_optimized_ami" {
  name = "/aws/service/ecs/optimized-ami/amazon-linux-2/recommended"
}

data "aws_subnet" "this" {
  id = tolist(var.subnet_ids)[0]
}

data "aws_vpc" "this" {
  id = data.aws_subnet.this.vpc_id
}
