# Single Sign-On (SSO)

## What
With SSO, users authenticate against an identity provider (IdP), like GitHub, Forgejo, Keycloak, Microsoft Entra ID, instead of 
against each application's own user directory. The applications trust the IdP's confirmation, no longer hold credentials themselves, 
and your team signs in everywhere on the platform with one account.

Kubara helps you set up SSO for the components it deploys:

- Argo CD
- Grafana, Prometheus, Alertmanager (kube-prometheus-stack)
- Homer dashboard
- and potentially further components in the future

## Why
There is no single set of instructions for this. Every application solves SSO differently. 

Argo CD speaks OIDC, Grafana brings its own OAuth configuration, Prometheus and Homer have no authentication at all and sit behind oauth2-proxy.
Additionally not every provider speaks the same protocol — So some need a dedicated connector.

## How
This section provides configuration examples for some common Providers.



