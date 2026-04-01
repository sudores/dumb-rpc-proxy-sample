locals {
  service_name   = "rpc-proxy"
  container_name = "rpc-proxy"
  container_port = 8080
}

resource "aws_ecs_service" "this" {
  name                   = local.service_name
  cluster                = var.cluster_id
  task_definition        = aws_ecs_task_definition.this.arn
  launch_type            = "FARGATE"
  desired_count          = var.desired_count
  enable_execute_command = true

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = [aws_security_group.this.id]
    assign_public_ip = false
  }

  load_balancer {
    container_name   = local.container_name
    container_port   = local.container_port
    target_group_arn = aws_lb_target_group.this.arn
  }
}

resource "aws_lb_target_group" "this" {
  name        = "${var.name}-rpc-proxy-tg"
  port        = local.container_port
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.this.id
  target_type = "ip"

  health_check {
    protocol            = "HTTP"
    port                = local.container_port
    path                = "/health"
    interval            = 30
    timeout             = 10
    healthy_threshold   = 3
    unhealthy_threshold = 5
  }

  tags = merge(var.tags, {
    "com.amazon.ecs.container-name" = local.container_name
  })
}

resource "aws_ecs_task_definition" "this" {
  family                   = "${terraform.workspace}-${local.service_name}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn
  task_role_arn            = aws_iam_role.ecs_task_role.arn

  container_definitions = jsonencode([
    {
      name  = local.container_name
      image = var.image
      portMappings = [
        { containerPort = local.container_port, hostPort = local.container_port }
      ]
      environment = [
        { name = "UPSTREAM_URL", value = var.upstream_url },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.this.name
          awslogs-region        = data.aws_region.this.id
          awslogs-stream-prefix = local.container_name
        }
      }
    }
  ])

  tags = merge(var.tags, var.app_version != null ? { Version = var.app_version } : {})
}

resource "aws_security_group" "this" {
  name        = "${local.service_name}-${var.name}"
  description = "Security group for service ${local.service_name}"
  vpc_id      = data.aws_vpc.this.id
  tags        = var.tags
}

resource "aws_vpc_security_group_ingress_rule" "this" {
  for_each = {
    for i, v in var.ingress_rules : tostring(i) => v
  }

  security_group_id = aws_security_group.this.id
  cidr_ipv4         = each.value.cidr_ipv4
  from_port         = each.value.from_port
  to_port           = each.value.to_port
  ip_protocol       = each.value.ip_protocol
  description       = each.value.description
}

resource "aws_vpc_security_group_egress_rule" "all" {
  security_group_id = aws_security_group.this.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/ecs/${var.name}/${local.service_name}"
  retention_in_days = 30
  tags              = var.tags
}

resource "aws_iam_role" "ecs_task_role" {
  name = "ecsTaskRole-${var.name}-${local.service_name}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role" "ecs_task_execution_role" {
  name = "ecsTaskExecutionRole-${var.name}-${local.service_name}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_role" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy_attachment" "ecs_task_role" {
  role       = aws_iam_role.ecs_task_role.name
  policy_arn = aws_iam_policy.ecs_exec.arn
}

resource "aws_iam_policy" "ecs_exec" {
  name        = "${var.name}-ECSExec-${local.service_name}"
  description = "Minimum permissions for ECS Exec for ${local.service_name}"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "ssmmessages:CreateControlChannel",
        "ssmmessages:CreateDataChannel",
        "ssmmessages:OpenControlChannel",
        "ssmmessages:OpenDataChannel",
      ]
      Resource = "*"
    }]
  })
}

data "aws_region" "this" {}
data "aws_subnet" "this" {
  id = tolist(var.subnet_ids)[0]
}
data "aws_vpc" "this" {
  id = data.aws_subnet.this.vpc_id
}
