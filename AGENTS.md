# Inventory Service Agent Context

## What this project is
ระบบinventoryจัดการอุปกรณ์ในแผนกเซอร์วิสของบริษัทอีซี่บาย: เพิ่มอุปกรณ์, ดูข้อมูล,เบิก-คืน, แจ้งซ่อม, นำเข้าข้อมูล, รายงาน
ใช้ภายในองค์กรเท่านั้น ไม่ใช่ระบบขายสินค้า
กฎเหล็ก: อุปกรณ์แต่ละชิ้นมีรหัส Asset Name ที่ต่างกัน 1 คำขอเบิกเลือกอุปกรณ์ได้ 1 ชิ้นเท่านั้น ห้ามยืมซ้ำรหัส asset name เดียวกันเด็ดขาด (equipment ที่มีคำขออยู่ในสถานะ 'รออนุมัติ'/'อนุมัติ'/'รอตรวจรับคืน'/'เกินกำหนดคืน' แล้ว จะสร้างคำขอใหม่ซ้ำไม่ได้ — บังคับด้วย partial unique index `one_active_borrow_per_equipment` บนตาราง `borrow_records`)

## Tech stack
- Backend: Go 1.22+, Gin, PostgreSQL ผ่าน Docker (ไม่ใช่ SQLite)
- Database GUI:   docker-compose up -d pgadmin   (เปิดที่ http://localhost:5050)
- Backend tests: go testing + net/http/httptest + testcontainers-go (ephemeral Postgres ต่อ test)
- Frontend: React + Tailwind (มีอยู่แล้วใน /frontend)
- API contract: docs/openapi.yaml คือแหล่งความจริงเดียว
- Lint/Security: golangci-lint, gosec, govulncheck / ESLint, tsc

## Project layout
- backend/cmd/server/       main.go
- backend/internal/handler/ HTTP layer (Gin) รู้จักแค่ HTTP
- backend/internal/service/ business rules
- backend/internal/repo/    SQL เท่านั้น
- backend/internal/db/      connection + schema
- frontend/src/             React app (api/, components/, pages/)
- docs/                     SPEC.md, PLAN.md, TASKS.md, openapi.yaml ← อ่านก่อนเริ่มงานทุกครั้ง

## Rules (must follow)
1. Plan ก่อน code ห้ามเขียนโค้ดก่อนเสนอแผนสั้น ๆ ให้ human เห็น
2. ทำทีละ 1 task จาก docs/TASKS.md เท่านั้น ห้ามทำเกินขอบเขต task
3. Test ต้องมีก่อนหรือพร้อมโค้ดเสมอ และห้ามลบ/แก้ test เพื่อให้ผ่าน
4. Business rule บังคับใช้ที่ database constraint ไม่ใช่แค่ใน application code
5. ทุก endpoint ต้องตรงกับ docs/openapi.yaml ถ้าต้องเปลี่ยน ให้แก้ contract ก่อนแล้วถาม human
6. Error response ใช้รูปแบบเดียวทั้งระบบ: {"error": {"code": "...", "message": "..."}}
7. มีคำถามหรือความกำกวม ให้ถาม human ก่อน ห้ามเดา

## Commands
- Run backend:  cd backend && go run ./cmd/server   (port 8080)
- Backend test: cd backend && go test ./...
- Backend lint: cd backend && golangci-lint run && gosec ./... && govulncheck ./...
- Run frontend: cd frontend && npm start         (port 3000)
- Frontend check: cd frontend && npm run lint && npm run build
- Database up:    cd backend && docker-compose up -d --build
- Migration test: cd backend && go test ./internal/db/...

## ข้อควรรู้ PostgreSQL สำหรับ AI (ต่างจาก SQLite)
- FK enforcement เปิดอยู่โดย default เสมอ ไม่ต้องตั้งค่าเพิ่ม แต่ต้องระวังลำดับ INSERT/DELETE
- ใช้ MVCC ไม่มีปัญหา "database is locked" — สำหรับ AC-4 ให้พึ่ง unique constraint violation (23505) จับ race condition
- ตั้ง `statement_timeout` (เช่น 5000ms) ทุก connection
- Auto-increment ใช้ `SERIAL`, timestamp ใช้ `TIMESTAMPTZ` เสมอ
- testcontainers-go (T-02) ให้ container แยกต่อ test อยู่แล้ว แต่ spin up/teardown ช้ากว่า SQLite in-memory
- Enum สถานะทั้งหมดในระบบเก็บเป็น**ภาษาไทย** (ผ่าน CHECK constraint ไม่ใช่ Postgres ENUM type):
  - `equipment.status`: `'กำลังใช้งาน'`, `'ในคลัง'`, `'เสียหาย'`, `'เลิกใช้งาน'`
  - `borrow_records.status`: `'รออนุมัติ'`, `'อนุมัติ'`, `'ไม่อนุมัติ'`, `'รอตรวจรับคืน'`, `'คืนแล้ว'`, `'เกินกำหนดคืน'`
  - `repair_records.status`: `'ส่งซ่อม'`, `'กำลังซ่อม'`, `'ซ่อมเสร็จ'`, `'ซ่อมไม่ได้'`
  - อย่าใช้ชื่อ enum ภาษาอังกฤษ (pending_approval/approved/... หรือ pending/in_progress/completed/failed) ในโค้ด/ทดสอบที่ query DB ตรง ๆ — ใช้ได้เฉพาะตอน map เป็น API response/internal state name เท่านั้น