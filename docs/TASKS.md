# TASKS Inventory Service MVP
ทำทีละ task ตามลำดับ ห้ามข้าม ห้ามรวม scope
Endpoint ใช้ตาม API Contract ใน SPEC เท่านั้น

## Phase 0 Contract
T-01 สร้าง docs/openapi.yaml ครบ 20 endpoints ตาม SPEC
     DoD: paths และ status/error codes ตรง SPEC ทุก endpoint และผ่าน validator; เป็นแหล่งอ้างอิงของทุก task ต่อไป

## Phase 1 Walking skeleton
T-02 Test: database schema (ephemeral Postgres ต่อ test ผ่าน testcontainers-go, test helper package) — ครอบ equipment.status 4 ค่า, borrow_records.status 6 ค่า, borrow_records มี created_by_user_id FK + borrower_name text NOT NULL (ไม่มี start_date/requester_id/borrower_id/borrower_department เดิม), equipment_types table (id, name, min_quantity) พร้อม seed data, activity_logs มี resource_type/resource_id/metadata, partial unique index กัน AC-4
     DoD: go test ./internal/db/... รันแล้วแดง ครอบเงื่อนไข schema ครบทุกจุด; testcontainers สร้าง Postgres ชั่วคราวสำเร็จ (ยังไม่มี migration ให้ผ่าน)

T-03 Implement: database migration package ตาม T-02 (open Postgres, migrate, constraint AC-4, seed equipment_types), อัปเดต AGENTS.md Commands
     DoD: go test ./internal/db/... ผ่าน; migration รันผ่านไม่มี error และ teardown อัตโนมัติ; schema ตรง SPEC ทุกจุด; equipment_types มี seed data ครบทุก type ที่ใช้งานจริง

T-03b Dev environment: docker-compose สำหรับ dev PostgreSQL (แยกจาก ephemeral test instance) + pgAdmin preconfigure connection อัตโนมัติ, อัปเดต AGENTS.md Commands
     DoD: docker-compose up -d --build ได้ container dev DB + pgAdmin; เปิด http://localhost:5050 เห็น dev database ทันทีไม่ต้องตั้งค่าเอง; ไม่กระทบ test tooling ของ T-02/T-03

T-04 Test: server bootstrap + GET /health + error format กลาง
     DoD: go test รันแล้วแดง ครอบ success/error shape ของ walking skeleton

T-05 Implement: Gin server + PostgreSQL bootstrap + /health + error mapping helper
     DoD: go test ผ่าน; health คืน 200; error response รูปแบบเดียวกันทั้งระบบ; go run ./cmd/server สตาร์ท port 8080 ได้

T-06 Test: activity log helper (insert user_id, action, resource_type, resource_id, metadata) — ใช้ร่วมทุก phase ถัดไป
     DoD: go test รันแล้วแดง ครอบ input ครบ

T-07 Implement: activity log helper ตาม T-06
     DoD: go test ผ่านรวม T-06; helper เป็น path เดียวที่เขียน activity_logs

## Phase 2 Equipment
T-08 Test: equipment create/list/update/delete + permissions (AC-1, AC-2, AC-10) รวม activity_logs (AC-9)
     DoD: go test รันแล้วแดง ครอบ success/validation/duplicate/forbidden/AC-9

T-09 Implement: equipment CRUD + permissions ตามบทบาท, เขียน activity_logs ทุก mutation
     DoD: go test ผ่าน; endpoint ตรง API Contract

T-10 Test: UI checklist หน้าอุปกรณ์ (loading/error/empty/success) + ฟอร์มสร้าง/แก้ไข
     DoD: checklist ครบ; ใช้ typed API client เท่านั้น ไม่เรียก fetch ตรง

T-11 Implement: React pages ดู/สร้าง/แก้ไขอุปกรณ์ ด้วย typed API client
     DoD: checklist ผ่านครบ; ข้อมูลตรงกับ backend

## Phase 3 Borrow requests
T-12 Test: create/list/detail + duplicate protection (AC-3, AC-4, AC-2b) — role User เท่านั้นสร้างได้ (created_by_user_id auto จาก context, ไม่รับจาก body), borrower_name รับจาก request body (required, ว่างเปล่า → 422), Admin/System Admin POST → 403, equipment ไม่พบ → 404, invalid borrow type, unavailable, duplicate (รวม concurrent test เช็ค partial unique index), filter query param (status/equipment_id/borrower_name), activity_logs (AC-9)
     DoD: go test รันแล้วแดง ครอบทุกกรณี รวม 403 ของ Admin/System Admin, 422 เมื่อ borrower_name ว่าง, filter 3 ตัว, AC-9; ไม่มี test BORROWER_NOT_FOUND/requester_id/borrower_department (ตัด concept "borrower เป็น user account" ทิ้งแล้ว)

T-13 Implement: create/list/detail endpoint — บังคับ role User เท่านั้น (Admin/System Admin → 403), created_by_user_id จาก authenticated context เสมอ, รับ borrower_name (required) จาก request body ตรง ๆ ไม่ validate เป็น FK, กฎซ้อน/สถานะเริ่มต้น/constraint ใน DB, filter query param (status/equipment_id/borrower_name), เขียน activity_logs ผ่าน helper
     DoD: go test ผ่าน; response ตรง API Contract (มี created_by_user_id/borrower_name, ไม่มี requester_id/borrower_id/borrower_department เดิม, POST body ไม่รับ created_by_user_id จาก client); เฉพาะ role User สร้างคำขอได้จริง

T-14 Test: approve/reject (AC-5, AC-5b) — สำเร็จ, ซ้ำ/ไม่ pending → 409, User → 403, activity_logs
     DoD: go test รันแล้วแดง ครอบ AC-5/AC-5b/AC-9

T-15 Implement: PUT .../approve, PUT .../reject, activity_logs
     DoD: go test ผ่าน; transition สถานะตรง SPEC

T-16 Test: return/confirm-return (AC-6, AC-6b) — สำเร็จ, คืนไม่ได้ → 409, User คืนของคนอื่น → 403, ยืนยันไม่ pending-return → 409, activity_logs
     DoD: go test รันแล้วแดง ครอบ AC-6/AC-6b/AC-9

T-17 Implement: POST .../return, PUT .../confirm-return, activity_logs
     DoD: go test ผ่าน; borrow lifecycle ตรง SPEC

T-18 Test: overdue detection (AC-6c) — temporary + approved + current_date > return_date → เกินกำหนดคืน ตอน query
     DoD: go test รันแล้วแดง ครอบเงื่อนไขวันที่ครบ

T-19 Implement: derived-status logic ใน repo/service query layer
     DoD: go test ผ่าน; overdue ทำงานตาม SPEC

T-20 Test: UI checklist ดู/สร้าง/อนุมัติ/ปฏิเสธ/คืน/รับคืน — ปุ่ม "คืนอุปกรณ์" (User) กับ "รับคืน" (Admin/System Admin) แยกกัน, ไม่แสดงปุ่ม/ฟอร์มสร้างคำขอให้ Admin/System Admin เห็น
     DoD: checklist ครบทุก action; typed API client เท่านั้น

T-21 Implement: React pages ดู/สร้างคำขอ (เฉพาะ role User, ไม่มี field เลือก borrower), อนุมัติ/ปฏิเสธ (Admin/System Admin), คืน/รับคืน (ปุ่มแยกตาม role)
     DoD: checklist ผ่านครบ; ข้อมูลตรงกับ backend

## Phase 4 Repair tickets
T-22 Test: repair ticket lifecycle (AC-7, AC-2c, AC-8) — สร้างสำเร็จ, ไม่พบ → 404, เลิกใช้งาน → 409, list/detail, completed/failed/invalid enum/closed/role check, activity_logs ทั้ง create และ status update
     DoD: go test รันแล้วแดง ครอบ AC-7/AC-2c/AC-8/AC-9

T-23 Implement: POST /repair-tickets, GET list/detail, PUT .../status, state transition อุปกรณ์อัตโนมัติ, activity_logs
     DoD: go test ผ่าน; endpoint + equipment state ตรง API Contract/AC

T-24 Test: UI checklist ดู/สร้าง/อัปเดตสถานะใบแจ้งซ่อม
     DoD: checklist ครบ; ปุ่มอัปเดตสถานะเห็นเฉพาะ Admin/System Admin

T-25 Implement: React pages ดู/สร้างใบแจ้งซ่อม + อัปเดตสถานะ (Admin/System Admin เท่านั้น)
     DoD: checklist ผ่านครบ; ข้อมูลตรงกับ backend

## Phase 5 Import and reporting
T-26 Test: POST /import (CSV) — header row บังคับ, validation/duplicate, response {total, success, failed}, activity_logs
     DoD: go test รันแล้วแดง ครอบ success/error ครบ

T-27 Implement: POST /import, activity_logs
     DoD: go test ผ่าน; response ตรง SPEC

T-28 Test: GET /dashboard (System Admin 4 KPI global รวม equipment_below_minimum, Admin 5 KPI global รวม equipment_below_minimum, User 4 KPI scope เฉพาะรายการที่ created_by_user_id = authenticated user เท่านั้น + cross-user isolation — equipment_below_minimum คำนวณจาก COUNT อุปกรณ์สถานะ "ในคลัง" ต่อ type เทียบกับ equipment_types.min_quantity ที่ seed ไว้), GET /reports (ทุก type, Admin/System Admin เท่านั้น, User → 403, from/to กรองด้วย created_at), GET /activity-logs (เรียงล่าสุด)
     DoD: go test รันแล้วแดง ครอบทั้ง 3 endpoint, scope ด้วย created_by_user_id/global count, role restriction, filter; test equipment_below_minimum ครอบกรณี type ที่ต่ำกว่า/เท่ากับ/สูงกว่า min_quantity

T-29 Implement: GET /dashboard (global สำหรับ Admin/System Admin รวม equipment_below_minimum, scope ด้วย created_by_user_id สำหรับ User ทั้ง 4 KPI), GET /reports (บังคับ Admin/System Admin, filter created_at), GET /activity-logs
     DoD: go test ผ่าน; response ตรง SPEC (ไม่มี requester_id/start_date/borrower_id เดิม); dashboard scope ด้วย created_by_user_id ถูกต้อง; equipment_below_minimum คำนวณจาก equipment_types.min_quantity (ไม่มี endpoint แก้ไขค่านี้ใน MVP); reports บังคับ role ถูกต้อง

T-30 Test: UI checklist นำเข้าข้อมูล (เมนูแยก), แดชบอร์ด, รายงาน, activity-logs
     DoD: checklist ครบทุกหน้า; ไม่มี component เรียก fetch ตรง

T-31 Implement: React pages นำเข้าข้อมูล (Admin/System Admin), แดชบอร์ด, รายงาน (Admin/System Admin), activity-logs
     DoD: checklist ผ่านครบ; ข้อมูลตรงกับ backend

## Phase 6 Hardening
T-32 Quality gates บนเครื่อง: golangci-lint run && gosec ./... && govulncheck ./... และ npm run lint && npm run build
     DoD: ผ่านครบทั้ง 4 คำสั่ง exit code 0 ไม่มี finding ค้าง

T-33 CI ด้วย GitHub Actions (build/test/lint backend+frontend) รันอัตโนมัติบน PR
     DoD: workflow เขียวบน branch ที่ตั้งไว้

T-34 Cross-agent review ด้วย session ใหม่ เทียบ spec/plan/tasks/implementation/test coverage
     DoD: มี review report; issues บันทึกเป็น task ตามมา ไม่แก้เงียบ ๆ ใน task นี้