# Demo V1 Release Iter00001 Proofcheck

Minimal GoShip starter. Add modules with `ship module:add`.

Fresh-app loop:
- `ship db:migrate`
- `go run ./cmd/web`
- `ship verify --profile fast`

Included by default:
- auth routes
- profile routes
- landing page
- home feed page

Excluded by default:
- payments
- push notifications
- PWA install flow
