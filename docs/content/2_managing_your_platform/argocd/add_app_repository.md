# How to add a repository to Argo CD
To deploy applications with Argo CD we need to connect Repositories. This is also useful if you want to give developers
a way to deploy their applications onto your platform without much intervention on your side.

An Argo CD  App Repository is a logical concept to control where applications may be deployed from.
For more information check:
https://argo-cd.readthedocs.io/en/stable/user-guide/private-repositories/

## **Add credentials to vault**
Add the repository credentials to your vault.
The example below uses HTTPS username + password/PAT authentication.
For most Git providers, `PAT` means Personal Access Token and is often tied to a user account.
For platform automation, prefer a technical or machine account instead of a personal user account.
Set `username` to the account name expected by your Git provider; the exact value is provider-dependent.
The helper shown on this page currently covers HTTPS username + password/PAT repositories.
For SSH deploy keys or GitHub App authentication on additional app repositories, create the Argo CD repository Secret manually according to the Argo CD documentation until the helper supports those modes as well.

```json
{
  "repo_pat": {
    "pat": "<repository password or PAT>"
  }
}
```
## **Modify Argo CD overlays**
Add the following to your `argo-cd/values.yaml`.
```yaml
repositories:
    - name: user-repo-mock
      projectScope: k8s-spoke-0
      # # This points to the secret in vault
      remoteRef:
        remoteKey: repo_pat
        remoteKeyProperty: pat
      repoType: git
      secretStoreRef:
        kind: ClusterSecretStore
        name: hub-0-production
      url: <the repo url you want to add>
      username: <the username for connection. also needed for PAT>
```

That whats happening behind the scenes:

![Add Repository](../../images/add-repository.png)


## **Push your changes to git**
Do not forget to push your changes to the git repository that serves your Argo CD application.
If you let Argo CD manage itself, it will add the configured repository to your cluster.
