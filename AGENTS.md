# Inventory Service Agent Context

## What this project is
ระบบinventoryจัดการอุปกรณ์ในแผนกเซอร์วิสของบริษัทอีซี่บาย: เพิ่มอุปกรณ์, ดูข้อมูล,เบิก-คืน, แจ้งซ่อม, นำเข้าข้อมูล, รายงาน
ใช้ภายในองค์กรเท่านั้น ไม่ใช่ระบบขายสินค้า
กฎเหล็ก: อุปกรณ์แต่ละชิ้นมีรหัส Asset Name ที่ต่างกัน 1 คำขอเบิกเลือกอุปกรณ์ได้หลายชิ้นพร้อมกัน ห้ามยืมซ้ำรหัส asset name เดียวกันเด็ดขาด

## Tech stack
- Backend: Go 1.22+, Gin, PostgreSQL ผ่าน Docker (ไม่ใช่ SQLite)
- Database GUI: pgAdmin
- Backend tests: go testing + net/http/httptest
- Frontend: React + Tailwind (มีอยู่แล้วใน /frontend)
- API contract: docs/openapi.yaml คือแหล่งความจริงเดียว
- Lint/Security: golangci-lint, gosec, govulncheck / ESLint, tsc

## Project layout
- backend/cmd/server/       main.go
- backend/internal/handler/ HTTP layer (Gin) รู้จักแค่ HTTP
- backend/internal/service/ business rules
- backend/internal/repo/    SQL เท่านั้น
- backend/internal/db/      connection + schema
- backend/database/init.sql    schema + seed data (รันตอน Docker สร้าง container ครั้งแรก)
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