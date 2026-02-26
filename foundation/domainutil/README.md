# domainutil

Helpers for revision-based consistency and UTC handling.

## Functions

- `IsUTC(time.Time) bool` - checks strict `time.UTC` location
- `CloneTimePtrUTC(*time.Time) *time.Time` - deep copy pointer and normalize to UTC
- `UTCOrZero(time.Time) time.Time` - normalize non-zero values to UTC
- `NextRevisionState(updatedAt, revision, at)` - monotonic revision/time progression with server-time ceiling
- `NextRevisionStateWithCeiling(updatedAt, revision, at, ceiling)` - deterministic clamp for client time skew
- `RequireRevision(current, expected)` - CAS-style revision guard

## Errors

- `ErrInvalidExpectedRevision`
- `ErrRevisionConflict`

`RequireRevision` returns typed errors with details:

- `InvalidExpectedRevisionError{Expected}`
- `RevisionConflictError{Current, Expected}`

Use `errors.Is` with sentinels for compatibility and `errors.As` to extract fields.

## Recommended Service Integration (Production)

Use a compare-and-swap (CAS) write in your repository and treat client time as untrusted.

```go
// 1) Validate expected revision from request
if err := domainutil.RequireRevision(current.Revision, cmd.ExpectedRevision); err != nil {
	if errors.Is(err, domainutil.ErrInvalidExpectedRevision) {
		return apperr.BadRequest(err)
	}
	if errors.Is(err, domainutil.ErrRevisionConflict) {
		return apperr.Conflict(err)
	}
	return err
}

// 2) Build next state with monotonic UTC timestamp
// Prefer explicit ceiling for deterministic behavior in services/tests.
nextAt, nextRev := domainutil.NextRevisionStateWithCeiling(
	current.UpdatedAt,
	current.Revision,
	cmd.ClientAt,
	clock.Now().UTC(),
)

// 3) Persist with CAS condition in DB
// UPDATE ... SET updated_at = $1, revision = $2 WHERE id = $3 AND revision = $4
rows, err := repo.UpdateWithCAS(ctx, current.ID, nextAt, nextRev, current.Revision)
if err != nil {
	return err
}
if rows == 0 {
	return domainutil.ErrRevisionConflict
}
```

## Business Examples

- **Order updates from web and mobile at the same time**: CAS prevents one channel from silently overwriting the other.
- **Client device clock is 2 days ahead**: `NextRevisionStateWithCeiling(..., clock.Now().UTC())` prevents future `updated_at` that would break sorting/SLA dashboards.
- **High-volume entities with long history**: revision saturation at `math.MaxInt64` avoids overflow rollover.

## Implementation Checklist

- Use `errors.Is` and `errors.As` (not `err == ...`).
- Always normalize persisted timestamps to UTC.
- Keep CAS at the storage layer (`WHERE revision = expected`).
- Prefer server clock for the time ceiling.

## Compatibility Note

`IsUTC` is strict and accepts only `time.UTC` location.
If your integration still emits offset-zero custom zones (for example `UTC0`),
use an observe-then-enforce rollout at service boundaries.
