# Add SSO Configuration (GitHub)

How to Setup your GitHub for SSO with kubara

To enable full Single Sign-On experience, you’ll need:

1. A GitHub **Organization** and at least one **Team**
2. Create **three GitHub OAuth Apps** under your org's Developer Settings → OAuth Apps:

### 1. Argo CD SSO

* **Homepage URL:** `https://cp.<your-domain>.stackit.run/argocd`
* **Callback URL:** `https://cp.<your-domain>.stackit.run/argocd/api/dex/callback`

### 2. Grafana SSO

* **Homepage URL:** `https://cp.<your-domain>.stackit.run/grafana`
* **Callback URL:** `https://cp.<your-domain>.stackit.run/grafana/login/github`

### 3. OAuth2 Proxy SSO

* **Homepage URL:** `https://cp.<your-domain>.stackit.run`
* **Callback URL:** `https://cp.<your-domain>.stackit.run/oauth2/callback`

!!! tip
    Save the **Client ID** and generate a **Client Secret** for each app – you'll need them later.

For other providers, consult their documentation to find the correct URLs and settings.
Here are some frequently used ones:

- [Microsoft Entra ID](https://learn.microsoft.com/en-us/entra/identity-platform/quickstart-register-app)
- [GitHub](https://docs.github.com/en/developers/apps/building-oauth-apps/creating-an-oauth-app)
- [Auth0](https://auth0.com/docs/quickstart/webapp/nextjs/01-login)
- [Google Cloud Identity](https://developers.google.com/identity/protocols/oauth2/native-app)
- [Okta](https://developer.okta.com/docs/guides/build-sso-integration/openidconnect/main/)
- [Forgejo (Gitea)](https://docs.gitea.com/development/oauth2-provider)