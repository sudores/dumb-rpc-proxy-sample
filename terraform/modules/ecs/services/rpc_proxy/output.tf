output "lb_rules" {
  description = "ALB listener rules for the rpc-proxy service"
  value = {
    rpc_proxy = {
      priority = 10
      actions = [{
        type             = "forward"
        target_group_arn = aws_lb_target_group.this.arn
      }]
      conditions = [
        { host_header = { values = [var.domain] } },
        { path_pattern = { values = ["/*"] } },
      ]
    }
  }
}
