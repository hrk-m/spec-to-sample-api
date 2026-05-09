---
name: Project Codebase Patterns
description: Key implementation patterns, naming conventions, and architecture decisions observed across sample-api and sample-front
type: project
---

## sample-api patterns

- Error responses use sentinel values: `domain.ErrBadParamInput` → 400, `domain.ErrNotFound` → 404, `domain.ErrInternalServerError` → 500
- Error message format: `{"message": "given param is not valid"}` / `{"message": "your requested item is not found"}` / `{"message": "internal server error"}`
- Query param parsing helpers are in `internal/rest/params.go`: `parseLimit`, `parseOffset`, `parsePathID`, `parseCommaSeparatedUint64`
- Dynamic SQL placeholders use `fmt.Sprintf` + `strings.Join` pattern (NOT NOT IN with empty slice — guarded with nil check)
- Domain types defined in `domain/` package; existing types (e.g., `domain.User`) are not modified when new types are introduced
- Mock files live in `group/mocks/` (service mocks) and `internal/rest/mocks/` (handler mocks)
- `offset` validation for `ListGroupMembers` is done only at the Handler layer (`parseOffset`), not in the Service layer (unlike `ListGroups` which validates offset in Service)

## sample-front patterns

- Feature-Sliced Design (FSD) v2.1 structure under `src/pages/group-detail/`
- API layer: `api/fetch-*.ts` files
- Model layer: `model/*.ts` and hooks
- UI layer: `ui/*.tsx` components + `ui/__tests__/` tests
- Cache pattern: module-level Map + listener Set for member list cache (`clearMemberListCache` triggers re-fetch via `refreshKey`)
- Debounce: 300ms for search query and filter changes
- `excludeGroupIds` is passed via ref to avoid stale closures in fetch callbacks

## list-group-members specific findings (2026-05-01)

- SQL uses sorted subquery inside `user_summary` CTE to work around MySQL 8.x not supporting ORDER BY inside JSON_ARRAYAGG
- `source_groups[0]` ordering guaranteed by `ORDER BY us.source_depth ASC, us.source_group_id ASC` in the subquery
- `exclude_group_ids` filter applied at `user_sources` CTE level with `WHERE d.root_child_id NOT IN (...)`
- Frontend `isDirectMember` and `buildSourceLabel` utility functions extracted to `model/group-detail.ts`
- `directMemberCount` computed in `useMemberList` hook and used for "全選択" logic in `MemberList.tsx`
