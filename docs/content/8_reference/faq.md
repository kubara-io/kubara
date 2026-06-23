# Frequently Asked Questions

## Is kubara only usable on STACKIT?

A clear and definite **NO**! kubara was specifically designed to be
provider-independent. kubara is based on Kubernetes and the underlying
infrastructure plays a secondary role that needs to be considered depending on
the use case.

---

## How much does it cost?

Even if kubara itself costs nothing, implementing incurs costs. The
exact costs depend on the setup and individual requirements and must be
calculated individually.
We recommend checking in advance which services are to be used to carry out
a rough calculation.

Services that may incur costs:

 - Work effort of the platform team
 - Hardware/Hyperscaler
 - DNS/PKI/Certificates
 - Artifactory
 - GIT
 - Key-Vaults
 - Network-Traffic
 - Storage
 - Individual support-contracts
 - ...

---

## What knowledge is required to use kubara

kubara includes a selection of standard tools, templates and best
practices for professionally building and operating a Kubernetes platform. It
is designed for users who are just starting out and want to build a Kubernetes
platform. Nevertheless, the following skills are necessary to understand and
use kubara:

1. **Linux Basics:** Understanding of basic `Linux` commands and operations.

2. **Following Documentation:** Ability to follow detailed technical
documentation and step-by-step guides.

3. **Version Control:** Basic experience with `Git` for managing code and
configurations.

4. **Container Fundamentals:** Basic knowledge of `containerization` and `Docker`.

5. **Kubernetes Concepts:** Familiarity with basic `Kubernetes` concepts like
`pods`, `services`, and `deployments`.

6. **Network Practices:** Understanding the basics of network communication
like `api`, `firewall` and `proxy`.

7. **Security Practices:** Understanding of basic security practices like
`authentication`, `authorization` and `policies`.

---

## How do i create a dockerconfig.json for .env-file

```bash
kubectl create secret docker-registry regcred
--docker-server=https://index.docker.io/v1/
--docker-username=YOUR_USER
--docker-password=YOUR_PASSWORD
--docker-email=optional@example.com
--namespace default
--dry-run=client -o yaml
```

---

## Can I use kubara in a production environment?

kubara covers as many use cases as possible. Since every environment
is unique, it's very important to know the requirements and check whether all
of them are met. If a requirement isn't met, don't worry! You always have the
option to customize things.
kubara will be expanded in the future to cover special requirements such
as high availability, high security requirements or scalable infrastructures.

---

## What prerequisites do I have?

Check [Prerequisites](../1_getting_started/prerequisites.md)

---

## Are the used tools OSS and can I use them in an enterprise environment?
The applications included in kubara are exclusively `open source`
products that can also be used in a corporate context. The license conditions
are regularly reviewed and taken into account when selecting products.

---

## Can I exchange tools and change configurations?

Of course, you can decide by yourself which tools and settings to use.
Predefined template structures allow you to standardize the use of tools
and resources that can be expanded at any time. kubara helps to avoid
unnecessary complexity and redundancies, e.g. of code or manifests.

---

## Can I deploy any application on the platform?

The platform is designed for container workloads only. In principle, any
container application can be deployed. To ensure secure and stable operation,
the application must meet the following requirements:
[The Twelve-Factor APP](https://12factor.net)

---

## When to Use App of Apps vs. ApplicationSets?

- `App of Apps` is a pattern where a single parent Argo CD Application manages
  multiple child Applications, typically defined in separate YAML files.

- `ApplicationSet` is a controller that dynamically creates multiple
  Applications based on generators (e.g. Git directories, cluster lists or
  matrixes).

Essentially, it is an architectural- rather than a functional decision.
Each serves a different purpose and can coexist in the same Argo CD setup.

- `App of Apps` is mainly used for a core platform setup
- `ApplicationSets` for scalable, templated application rollout

---

## What happens when OAuth2 Proxy is disabled?

If you choose to disable OAuth2 Proxy during setup, you skip the OAuth2 Proxy configuration steps as part of the bootstrapping guide.

Since OAuth2 Proxy provides authentication for ingress routes, as a security measure no traefik routes will be deployed. Without them none of your apps will be publicly reachable until you configure an alternative authentication mechanism and deploy the routes yourself.
