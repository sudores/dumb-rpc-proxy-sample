module "ecs" {
  source  = "terraform-aws-modules/ecs/aws"
  version = "~>5.12.0"

  cluster_name                          = var.name
  default_capacity_provider_use_fargate = false
  autoscaling_capacity_providers = {
    "${terraform.workspace}-EC2" = {
      auto_scaling_group_arn         = module.autoscaling.autoscaling_group_arn
      managed_termination_protection = "ENABLED"

      managed_scaling = {
        maximum_scaling_step_size = 2
        minimum_scaling_step_size = 1
        status                    = "ENABLED"
        target_capacity           = 70 // TODO: Fix according to docs https://aws.amazon.com/blogs/containers/deep-dive-on-amazon-ecs-cluster-auto-scaling/
      }
      default_capacity_provider_strategy = {
        weight = 60
      }
    }
  }


  tags = var.tags
}

locals {
  user_data = <<-EOT
        #!/bin/bash

        cat <<EOF >> /etc/ecs/ecs.config
        ECS_CLUSTER=${var.name}
        EOF
  EOT
}

module "autoscaling" {
  source                          = "terraform-aws-modules/autoscaling/aws"
  version                         = "~>8.2.0"
  name                            = var.name
  image_id                        = jsondecode(data.aws_ssm_parameter.ecs_optimized_ami.value)["image_id"]
  ignore_desired_capacity_changes = true

  vpc_zone_identifier = var.subnet_ids
  health_check_type   = "EC2"
  min_size            = 1
  max_size            = 4
  desired_capacity    = 2
  user_data           = base64encode(local.user_data)

  instance_type   = "t3.small"
  security_groups = [aws_security_group.this.id]
  key_name        = "ah-main"

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

data "aws_ssm_parameter" "ecs_optimized_ami" {
  name = "/aws/service/ecs/optimized-ami/amazon-linux-2/recommended"
}

resource "aws_security_group" "this" {
  name        = "asg-${var.name}"
  description = "Secutiry group for service asg"
  tags        = var.tags
  vpc_id      = data.aws_subnet.this.vpc_id
}

resource "aws_security_group_rule" "ingress" {
  type              = "ingress"
  security_group_id = aws_security_group.this.id
  cidr_blocks       = [data.aws_vpc.this.cidr_block]
  from_port         = 0
  protocol          = "tcp"
  to_port           = 65535
}

resource "aws_security_group_rule" "egress" {
  security_group_id = aws_security_group.this.id
  cidr_blocks       = ["0.0.0.0/0"]
  type              = "egress"
  from_port         = 0
  protocol          = "-1"
  to_port           = 65535
}

data "aws_region" "this" {}
data "aws_subnet" "this" {
  id = tolist(var.subnet_ids)[0]
}

data "aws_vpc" "this" {
  id = data.aws_subnet.this.vpc_id
}
