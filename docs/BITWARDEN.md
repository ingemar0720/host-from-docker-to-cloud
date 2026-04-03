# Bitwarden and this repo

Runtime and CI secret strategy is defined in **PLAN.md §8.3** (Bitwarden Secrets Manager projects + machine accounts, Zeabur bootstrap only).

This repository’s CLI does not call Bitwarden yet; use `bw` or the Secrets Manager API in your pipeline to inject values before `make deploy` or at container entry, as described in the plan.
