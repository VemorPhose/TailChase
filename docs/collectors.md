# Collector Interfaces

Collectors turn provider output into local Tailchase evidence. A collector should:

- implement `collect.ProviderCollector[Options]`
- return stable `ProviderMetadata` such as `github_actions` / `ci`
- preserve raw evidence under `.tailchase/runs/<run-id>/evidence/`
- return `Result.Sources` with provider metadata and raw paths
- return normalized `Result.Signals` only when the collector can do so safely
- leave common normalization to `bundle.Normalizer` when raw text is enough
- add tests with fixtures or fake clients, not live provider calls

GitHub Actions and GitLab CI follow this pattern. New CI providers should
validate config, collect raw evidence, record auditable artifact paths, and keep
provider-specific API details out of bundle generation.
