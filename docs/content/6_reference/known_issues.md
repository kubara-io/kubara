# Known Issues

This is a list of known issues and workarounds.
For a more complete list of issues for Kubara please take a look at the [Github Issues](https://github.com/kubara-io/kubara/issues).


## Secrets not being available with 

If SSO is enabled (`oauth2Proxy: enabled`), a syncing issue can occur, where ArgoCD is already up, while the External Secrets are not yet set available.

### Workaround

After External Secrets finished syncing restart Argo CD once:
```
$ kubectl -n argocd rollout restart deploy/argocd-server
$ kubectl -n argocd rollout status deploy/argocd-server
```
