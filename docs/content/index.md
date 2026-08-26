# kubara

![Status: opensource](https://img.shields.io/badge/status-opensource-lightgrey)
![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)
![Docs License: CC BY 4.0](https://img.shields.io/badge/docs%20license-CC%20BY%204.0-2ea44f)
![GitHub Discussions](https://img.shields.io/github/discussions/kubara-io/kubara)

## What is kubara?

### A package manager for your platform

Helm packages and deploys one application. Kubara applies that idea to an entire Kubernetes platform across a fleet of clusters.

A kubara catalog can bundle Helm charts, Terraform modules, scripts, and the whole GitOps structure needed for your platform. Kubara uses such a catalog to render reproducible manifests and bootstraps a central **hub cluster** that uses Argo CD to manage and reconcile identical platform environments across **hundreds of spoke clusters**.

Kubara is not an operator. It does not install a kubara controller or other kubara runtime components in your cluster. After bootstrap, Argo CD reconciles the generated platform state from Git but kubara pre-templates the cross-integration of all your platform components for you.

In short, kubara is a CLI for building, packaging, and bootstrapping Kubernetes platforms. Inspired by real-world needs at Schwarz Group to standardize and operate large-scale environments through pure GitOps.

That means you can not only bootstrap a platform with kubara, but also package, distribute, and reuse that platform setup across many clusters.


![Overview](assets/diagrams.drawio)

Commit the generated output to Git. Then use the same catalog and configuration model for another cluster.

## What kubara does?

- ⚙️ **Full Platform Bootstrap** - From infrastructure to observability, GitOps, and secrets
- 📦 **Platform Catalog Packaging** - Package, distribute, and reuse platform setups
- 🧱 **Modular by Design** - Helm and Terraform based components, easily extendable
- 🔁 **Multi-cluster Support** - Hub + N Spoke clusters, with app targeting
- 🚀 **Fast Setup** - Ready to go in under 30 minutes
- ☁️ **Cloud & Edge Ready** - Use it across cloud, hybrid, or bare-metal environments
- 🔐 **Built-in Best Practices** - Production-grade setup used by real-world platforms

## Why use kubara?

Building and maintaining a multi-cluster Kubernetes platform means keeping infrastructure, GitOps configuration, secrets, and shared components in sync.

kubara gives you one CLI that:

- Generates required configuration and secrets
- Renders Terraform and Helm outputs based on predefined templates
- Bootstraps your platform using Argo CD
- Packages catalogs for repeatable rollouts
- Allows for easy onboarding of new clusters and workloads

All based on real-world usage at Schwarz Group and the experience of multiple engineering teams - so you don't have to reinvent the wheel.

## How it works

The usual flow is:

1. 📄 Initial configuration via `.env` and `config.yaml`
2. 🧩 Rendering and deploying Terraform and Helm modules using `kubara generate`
3. 📦 Commit the output to your GitOps repository.
4. 🚀 Bootstrapping the Hub cluster and Argo CD using `kubara bootstrap`
5. 🐙 Let Argo CD reconcile the platform state from Git.
6. 🧱 Adding additional spoke clusters and workloads

## 🚀 Get started

Follow the [bootstrap guide](1_getting_started/bootstrapping.md) to:

- Install the CLI
- Prepare your `.env` and `config.yaml`
- Run `kubara init`, `kubara schema` and `kubara generate`
- Bootstrap Argo CD with `kubara bootstrap <cluster-name>`

## 📚 Videos, talks, and articles

- [📝 GitOps for 15,000+ Clusters: What Large-Scale Testing with vCluster Taught Us | Blog at Medium and ITNEXT, 2026](https://medium.com/itnext/gitops-for-15-000-clusters-what-large-scale-testing-with-vcluster-taught-us-41e4b0d43e0b)
- [🎙️ One Platform Could Not Fit Them All | Virtual Talk at WeAreDevelopers, 2026 ](https://www.wearedevelopers.com/en/videos/1919/one-platform-could-not-fit-them-all)
- [🎙️ When Building One Platform Isn’t Enough | Talk at CloudLand, 2026 ](https://meine.doag.org/events/cloudland/2026/agenda/#agendaId.7200)
- [🎙️ Kubernetes Platform Blueprint | Co-Located Workshop with vCluster at KubeCon, 2026](https://www.vcluster.com/events/kubernetes-platform-blueprint)
- [🎥 Free Course based on kubara | GitOps for Platform Engineering at Platform University, 2026 ](https://university.platformengineering.org/introduction-to-gitops-for-platform-engineering)
- [🎙️ How to build a Multi-Tenancy Internal Developer Platform with GitOps and vCluster | Talk at ContainerDays Hamburg, 2025](https://www.youtube.com/watch?v=yQsnA91Gtcs)
- [🎥 How to build a multi-tenancy Internal Developer Platform with GitOps and vCluster | Virtual Workshop at PlatformCon, 2025](https://www.youtube.com/watch?v=2wQ4w1NKfd4)
- [🎥 Load Testing Argo CD at Scale with vCluster and GitOps | Webinar at vCluster, 2025 ](https://www.youtube.com/watch?v=0XEWn4VmiDE)
- [🎙️The GitOps Blueprint: Multi-Tenant Kubernetes with Argo CD in 30 Minutes | Talk at Cloud X Summit, 2025 ](assets/The_GitOps_Blueprint_Multi-Tenant_Kubernetes_with_Argo_CD_in_30_Minutes.pdf)
- [📝 How We Load Test Argo CD at Scale: 1,000 vClusters with GitOps on Kubernetes | Blog at Medium and ITNEXT, 2025 ](https://medium.com/itnext/how-we-load-test-argo-cd-at-scale-1-000-vclusters-with-gitops-on-kubernetes-d8ea2a8935b6)
- [📝 How to Build a Multi-Tenant Kubernetes Platform with GitOps and vCluster | Blog at Medium and ITNEXT, 2025](https://medium.com/itnext/from-ci-to-kubernetes-catalog-building-a-composable-platform-with-gitops-and-vcluster-7e1decaa81da)

## 🤝 Contributing

We would 💙 your contributions! Here's how to get started:

1. Check the [issues](https://github.com/kubara-io/kubara/issues).
2. Open an issue if yours is not listed.
3. Open a pull request if you want to fix it.
4. Follow the [commit message conventions](https://github.com/kubara-io/kubara/blob/main/CONTRIBUTING.md#commit-and-pr-guidelines).

## Versioning

kubara follows [Semantic Versioning](http://semver.org/), with releases named `v0.1.0-something`.
Releases are listed in the [Release section](https://github.com/kubara-io/kubara/releases).


## License

Documentation is licensed under [CC BY 4.0](https://github.com/kubara-io/kubara/blob/main/NOTICE.md#documentation-license).
Software source code is licensed under [Apache 2.0](https://github.com/kubara-io/kubara/blob/main/LICENSE).
