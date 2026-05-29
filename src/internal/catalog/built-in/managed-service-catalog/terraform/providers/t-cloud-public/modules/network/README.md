# Network

Creates T Cloud Public network primitives for the example CCE stack: VPC, subnet, optional NAT gateway and optional external load balancer with EIP.

The external load balancer defaults to shared ELB v2. Set `load_balancer_type = "dedicated"` to create a dedicated ELB v3 load balancer instead. Dedicated load balancers can be placed in one or more availability zones and use configurable L4/L7 flavor names.

Use this module directly for generated demo infrastructure, or replace it with existing VPC/subnet IDs when integrating into a pre-existing landing zone.
