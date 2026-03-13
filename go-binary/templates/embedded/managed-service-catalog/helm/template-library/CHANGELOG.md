# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.5] - 2026-02-11
### Added
- additional value file parsing "additional-values.yaml" in `_argo.appset.tpl_`. Can be used to override default values

## [0.0.4] - 2025-11-13
### Added
- sources and source.kustomize fields in `_argo.app.tpl_`
- sources field in `_argo.appset.tpl_`
### Removed
- managedNamespaceMetadata fields in `_argo.app.tpl_`

## [0.0.1] - 2025-04-28
### Added
- Initial blueprint version
