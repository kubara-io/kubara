# Known Issues

This is a list of known issues and workarounds.
For a more complete list of open issues for Kubara please take a look at the [Github Issues](https://github.com/kubara-io/kubara/issues) page.


## Bootstrap race condition between Argo CD and External-Secrets when using OAuth2 Proxy / SSO

If SSO is enabled via `oauth2Proxy: enabled`, it can happen that Argo CD and external-secrets are syncing 
simultaneously. Sometimes causing the Dex component of Argo CD to (re)start before the secret for the OAuth2 Proxy 
integration is ready. Which leads to a running pod without the correct credentials. 

### Workaround

After External-Secrets finished syncing the secrets, specifically the `oauth2-credentials` secret, restart Argo CD once:

```
# Secret should exist
$ kubectl -n argocd get secrets
NAME                             TYPE                             DATA
...
oauth2-credentials               Opaque                           2 
...

# A restart of Dex should be sufficient to get the integration working
$ kubectl -n argocd rollout restart deploy/argocd-dex-server

# Or simple force a quick restart of all components
$ kubectl -n argocd rollout restart sts,deploy 
```
