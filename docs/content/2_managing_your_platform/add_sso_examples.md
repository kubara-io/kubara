# SSO Examples: 
## Forgejo

**The following documentation provides configuration examples for SSO-Configuration   
with STACKIT GIT (Forgejo based).**  

These configurations are to enable kubara & Forgejo Users to quick-start with SSO.  

> ⚠️ **Prerequisites:**  
> Create Application in Forgejo for Grafana, OAuth2Proxy & Argo   
> [Forgejo Docs](https://forgejo.org/docs/next/user/oauth2-provider/#examples)    
>
> Same Callback URLs like GitHub  
>    [kubara Docs: Add SSO GitHub](add_sso.md)

With that being said, we can not provide Support for any SSO issues as you  might want to configure some parameters   
differently and the stable configuration might also be subject to change in the different Software-Projects.  

It's worth mentioning that Forgejo is a Fork of Gitea which aims to be compatible with GitHub.  
Sometimes you use Gitea Providers and sometimes its GitHub - **Keep that in mind when Updating and using this configuration in Production.**

Below are working examples when these Docs are released.

**They are based on:**  
- [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/configure-access/configure-authentication/github/)  
- [OAuth2Proxy](https://oauth2-proxy.github.io/oauth2-proxy/configuration/providers/gitea)  
- [Argo](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/#dex) & [Dex](https://dexidp.io/docs/connectors/gitea/)  

We recommend to take a look at them to make sure everything is configured as intended before  
you finally apply your SSO config.

### Forgejo SSO config for oauth2-proxy:

Change values in: **"kubara/customer-service-catalog/helm/your-cluster/oauth2-proxy/values.yaml"**
```yaml
oauth2-proxy:

    configFile: |-
      reverse_proxy = true
      redirect_url = "https://your-domain/oauth2/callback"
      email_domains = [ "*" ]
      cookie_secure = true
      upstreams = [ "file:///dev/null" ]
      github_org = "YOUR-ORGANIZATION"
      github_team = "Your-Team"
      provider = "github"
      scope = "read:user user:email read:org"
      login_url="https://your-forgejo.domain/login/oauth/authorize"
      redeem_url = "https://your-forgejo.domain/login/oauth/access_token"
      validate_url="https://your-forgejo.domain/api/v1/user/emails"
      provider_display_name="Your Forgejo"
```

Replace the values of the following config variables accordingly to your setup:

Replace "your-domain" in "redirect_url" with the value of "dnsName" set in kubara config.yaml and keep the trailing "/oauth2/callback".  

Replace values of  

- "login_url", "redeem_url" & "validate_url" with your according forgejo domain.
- "github_org" & "github_team" with your according org and team-names.
- "provider_display_name" with your desired name to be displayed on the SSO-Login-Button


---


### Forgejo SSO config for Grafana:   
(As part of Kube-Prometheus-Stack Helm-Chart)

### **Secrets**

The "grafana.ini" Parts get rendered into a Grafana-Config File and it's better to include your   
Secret  into your Container as an Environment-Variable. Env-Variables won't be touched in the rendering process 
which otherwise could lead to escaping-issues with special Characters, like backslashes, Dollarsign, Asterisk, Quotes, etc.

The below example includes the YAML identation-levels ("kube-prometheus-stack: grafana:") for reference.  

Change values in: **"kubara/customer-service-catalog/helm/your-cluster/kube-prometheus-stack/values.yaml"**

```yaml
kube-prometheus-stack:
  grafana:
    envFromSecrets:
      - name: oauth2-credentials
    envValueFrom:
      GF_AUTH_GENERIC_OAUTH_CLIENT_ID:      # desired name in pod env
        secretKeyRef:
          name: oauth2-credentials          # secret name
          key: client-id
      GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET:  # desired name in pod env
        secretKeyRef:
          name: oauth2-credentials          # secret name
          key: client-secret
    grafana.ini:
      auth.github:
        name: "Forgejo"
        enabled: true
        client_id: ${client-id}
        client_secret: ${client-secret}
        allow_sign_up: true
        auto_login: false
        scopes: "user:email,read:org"
        # Grafana doesn't support "allowed organizations" with Forgejo because of API-incompatibilites
        # neither with github-provider nor with the generic-oauth
        # allowed_organizations: "test-orga"
        # role_attribute_path: contains(groups[*], '@test-orga/Owners') && 'Admin' || 'Viewer'
        auth_url: https://your-forgejo.domain/login/oauth/authorize
        token_url: https://your-forgejo.domain/login/oauth/access_token
        api_url: https://your-forgejo.domain/api/v1/user
```
**Replace the values of the following config variables accordingly to your setup:**

Replace "your-forgejo.domain" in the following variables and keep the trailing paths: (like /login/...)

- auth_url
- token_url 
- api_url


---


### Forgejo SSO config for Argo:

Change values in: **"kubara/customer-service-catalog/helm/your-cluster/argo-cd/values.yaml"**

```yaml
  configs:
    cm:
      dex.config: |
        connectors:
          - type: gitea
            id: gitea
            name: Your Forgejo
            config:
              clientID: $oauth2-credentials:client-id
              clientSecret: $oauth2-credentials:client-secret
              baseURL: https://yourproject.git.onstackit.cloud
              orgs:
                - name: test-orga
                  teams:
                  - Owners        
      url: https://your-domain/argocd
    params:
      server.basehref: /argocd
      server.insecure: true
      server.rootpath: /argocd
    rbac:
      policy.csv: |
        g, test-orga:Owners, role:admin        
      policy.default: role:readonly
```
**Replace the values of the following config variables accordingly to your setup:**

* Replace "your-domain" in "configs.cm.url:" with your "dnsName" set in kubara config.yaml 
* Replace value of "name:" and "teams:" in "orgs.name:"
* Replace "baseURL: https://yourproject.git.onstackit.cloud" with your oauth2-Provider Domain (in this case: Your Forgejo).

And also replace "test-orga:Owners" according to your Organization Name and Team:

```yaml
      policy.csv: |
        g, test-orga:Owners, role:admin  
```

## Other

**Feel free to contribute further configurations that are verified as working.**
  
  
