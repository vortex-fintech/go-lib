# contactutil

Utilities for contact value normalization.

## Functions

- `NormalizeEmail(string) string` - trims and lowercases e-mail input
- `NormalizeE164(string) string` - trims phone value expected in E.164 format

## Notes

- These functions do **not** validate correctness.
- Validation should be performed in application/service layers.
