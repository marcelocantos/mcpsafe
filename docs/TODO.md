# TODO

## Features

- [ ] **Generic token provisioning**: Find a generic way for the service to
  allow adding tokens sourced from different sites. Currently the user must
  manually run `security add-generic-password` per backend. Consider a
  `mcpsafe add-token` subcommand or interactive setup flow that handles
  Keychain storage, possibly with site-specific OAuth flows or clipboard-based
  paste prompts.
