# Network

Creates T Cloud Public network primitives for the example CCE stack: VPC, subnet, optional NAT gateway, shared external load balancer, and optional independent dedicated load balancer.

The shared ELB v2 load balancer is enabled by default and is intended for cluster ingress. Set `enable_dedicated_load_balancer = true` to create an additional, independent dedicated ELB v3 load balancer. Dedicated load balancers can be placed in one or more availability zones and use configurable L4/L7 flavor names.

Use this module directly for generated demo infrastructure, or replace it with existing VPC/subnet IDs when integrating into a pre-existing landing zone.
