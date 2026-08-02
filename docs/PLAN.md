# PLAN Inventory Service MVP

## Backend foundation
- Build the backend around Gin with clear layers: handler → service → repo.
- Use PostgreSQL via Docker for persistence and enforce business rules in the service layer and database constraints.
- Implement error mapping from typed errors to HTTP status codes and the spec’s machine-readable error format.

## Core domain flows
- Implement equipment create/read/update/delete with role-based permissions and uniqueness rules for asset name and serial number.
- Implement borrow-request lifecycle: create, approve, reject, return, confirm-return, and overdue handling for temporary borrows.
- Implement repair-ticket lifecycle and automatic equipment state changes when a ticket is completed or failed.
- Persist all relevant actions to activity logs.

## Import and reporting
- Implement CSV import for equipment data and return summary results as total, success, and failed.
- Provide read endpoints for dashboard, reports, and activity logs aligned to the spec.

## Frontend
- Build React pages for equipment, borrow requests, repair tickets, dashboard, and reports.
- Keep components presentation-only and use a generated typed API client from docs/openapi.yaml rather than direct fetch calls.

## Testing
- Add backend tests with Go test and httptest using PostgreSQL-backed tests with one database per test.
- Validate main success and error cases so behavior matches the spec.

## ข้อควรรู้ PostgreSQL สำหรับ AI (ต่างจาก SQLite)
- FK enforcement เปิดอยู่โดย default เสมอ ไม่ต้องตั้งค่าเพิ่ม แต่ต้องระวังลำดับ INSERT/DELETE
- ใช้ MVCC ไม่มีปัญหา "database is locked" — สำหรับ AC-4 ให้พึ่ง unique constraint violation (23505) จับ race condition
- ตั้ง `statement_timeout` (เช่น 5000ms) ทุก connection
- Auto-increment ใช้ `SERIAL`, timestamp ใช้ `TIMESTAMPTZ` เสมอ
- testcontainers-go (T-02) ให้ container แยกต่อ test อยู่แล้ว แต่ spin up/teardown ช้ากว่า SQLite in-memory
