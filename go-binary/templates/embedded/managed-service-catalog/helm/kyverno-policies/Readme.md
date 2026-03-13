# Contents

This Helm Charts is composed of the official Kyverno-policies chart and additional custom policies.

# Usage
Deploy this chart onto a cluster with a running kyverno instance. The policies will be used by kyverno.

## CI
During development you can use kyverno cli to apply the policies to a manifests.
To overwrite variables and values that are needed to emulate how kyverno behaves on a live cluster you can utilize 
the files in **kyverno-policies/ci**.  
For example to test argo-cd charts against these policies:
1. Run helm template for argo-cd
    ```bash
   helm template charts/argo-cd --namespace test-namespace > template.yaml
   ```
2. Run helm template for kyverno-policies
    ```bash
    helm template charts/kyverno-policies --values charts/kyverno-policies/ci/override-issuer.yaml > policies.yaml
    ```
3. Run kyverno apply
    ```bash
    kyverno apply policies.yaml --resource template.yaml \
      --userinfo charts/kyverno-policies/ci/variables/user_info.yaml \
      --values-file charts/kyverno-policies/ci/variables/variables.yaml \
      --detailed-results -t
    ```

### Known issues/limitations
***failed to check deny conditions: failed to substitute variables in condition key: ... JMESPath query failed: Unknown key "policies_count" in path'***  
This may be caused by how the key is generated within the policy. Some keys rely on a apiCall against 
the k8s-api-server which we cannot emulate in CI.

Setting the namespace during helm template only sets the built-in variable ".Release.Namespace" but does not set a pseudo-k8s context.
Resources that does not specify their namespace by calling ".Release.Namespace" may have their namespace set to 
"default" when running helm template.


## IT-Grundschutz
### Base
#### disallowKubernetesSecrets

This Policy aims on restricting the usage of default Kubernetes Secrets.
Instead: the ExternalSecrets resource should be used in order to guarantee
that the secret is up to date and stored securely.
This policy ensures IT-Grundschutz-rules:
SYS.1.6.A8 as well as risk situations Kubernetes 2.2 and Container 2.7

if set: *you may avoid this policy by running your request as cluster-admin.*

#### enforceImageRegistry

This policy aims on disallowing the usage of insecure Images.
Only Images from internal registries guarantee security.
Therefore, any Deployment-spec and Pod-spec containing Images from
container-registries other than schwarzit.jfrog.io will be denied.
This policy ensures IT-Grundschutz-rules:
SYS.1.6.A6, SYS.1.6.A12, SYS.1.6.A13

if set: *you may avoid this policy by running your request as cluster-admin.*

### Standard
* createDefaultNetworkPolicy
* denyPortSsh
* manyContainersInPod
* requireLimits

#### createDefaultNetworkPolicy

This Policy creates a default Network Policy for every newly created namespace.
The Network-policy is of type "default-deny"
This policy ensures IT-Grundschutz-rules:
APP.4.4.A18

#### denyPortSsh

This Policy aims on denying ssh on local containers.
If SSH access is needed, an PolicyException needs to be defined!
This Policy ensures IT-Grundschutz-rule:
SYS.1.6.A16

if set: *you may avoid this policy by running your request as cluster-admin.*

#### manyContainersInPod

This Policy checks if a Pod has more than two containers defined (main and sidecar).
This policy ensures IT-Grundschutz-rule:
SYS.1.6.A11

if set: *you may avoid this policy by running your request as cluster-admin.*

#### requireLimits

This Policy denies any Pod or Pod-Controller not having Set Limits and Requests.
This Policy ensures IT-Grundschutz-Rules:
SYS.1.6.A15

if set: *you may avoid this policy by running your request as cluster-admin.*

## High
* deny-xxx
