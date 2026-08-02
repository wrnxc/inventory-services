# SPEC Inventory Service MVP

## Users
- System Admin: Full permission — จัดการอุปกรณ์ (เพิ่ม/ลบ/แก้ไข), อนุมัติ/ปฏิเสธคำขอเบิก, จัดการแจ้งซ่อม, ดูรายงานทุกประเภท, เห็น Dashboard 4 KPI(อุปกรณ์ทั้งหมด/อุปกรณ์ที่ถูกเบิก/อยู่ระหว่างซ่อม/อุปกรณ์ต่ำกว่าระดับขั้นต่ำ)
- Admin: เหมือน System Admin เกือบทั้งหมด, เห็น Dashboard 5 KPI(อุปกรณ์ทั้งหมด/คำขอเบิกรออนุมัติ/รายการคืนรอยืนยัน/รายการแจ้งซ่อมรอดำเนินการ/อุปกรณ์ต่ำกว่าระดับขั้นต่ำ)
- User (helpdesk/service staff): Read-only ต่ออุปกรณ์ + สร้างคำขอเบิก + กรอกฟอร์มแจ้งซ่อม — ไม่มีสิทธิ์แก้ไข/ลบอุปกรณ์, เห็น Dashboard 4 KPI (ทั้งหมด scope เฉพาะรายการที่ตนเองสร้าง/แจ้งเท่านั้น): คำขอรออนุมัติ/รายการเบิกสำเร็จ/อยู่ระหว่างซ่อม/คำขอที่กำลังดำเนินการ 
- ทั้งสามบทบาทเห็นรายการอุปกรณ์ชุดเดียวกัน สิทธิ์แก้ไขต่างกันตามบทบาทเท่านั้น ไม่มีมุมมองแยกตามบทบาท

## Out of scope (MVP v1)
- Authentication แบบ OAuth/SSO ภายนอก (ใช้ local user ที่สร้างไว้ให้แต่ละคนแทน)
- Multi-tenant ข้ามบริษัท (รองรับแค่สาขาในบริษัทเดียวกัน เช่น กรุงเทพ/โคราช/ระยอง)
- เมื่อระดับอุปกรณ์ประเภทไหนต่ำกว่าระดับขั้นต่ำ จะมีการแจ้ง notification ไปยัง system admin, admin
- Deployment

## Clarified implementation decisions for MVP
- การระบุตัวตนและสิทธิ์: ใช้ authenticated user context จาก local session/header ในระบบนี้ โดยถ้าไม่มี context ให้ตอบ 401 Unauthorized และถ้าไม่มีสิทธิ์ให้ตอบ 403 Forbidden
- สำหรับคำขอเบิก: มีเฉพาะ Role "User" (helpdesk) เท่านั้นที่สร้างคำขอเบิกได้ ระบบบันทึก created_by_user_id จาก authenticated context เสมอ (คือ helpdesk ผู้กรอกฟอร์ม); ผู้ครอบครองอุปกรณ์จริง (borrower_name) เป็น free-text ที่ helpdesk กรอกแทนพนักงานแผนกอื่น ไม่ใช่ user account ในระบบ; Admin/System Admin ไม่มีสิทธิ์สร้างคำขอเบิก มีหน้าที่แค่อนุมัติ/ปฏิเสธ
- State machine ของ borrow request ใช้สถานะต่อไปนี้: `pending_approval` → `approved` → `pending_return_confirmation` → `returned`; รองรับ `rejected` และ `overdue` เป็นสถานะเพิ่มเติม โดย transition ที่อนุญาตคือ pending_approval → approved/rejected, approved → pending_return_confirmation/overdue, pending_return_confirmation → returned
- เมื่อคำขออยู่ในสถานะ terminal เช่น `rejected` หรือ `returned` สามารถสร้างคำขอใหม่สำหรับอุปกรณ์เดิมได้อีกครั้ง; ในขณะที่อยู่ในสถานะ `pending_approval`, `approved`, `pending_return_confirmation`, `overdue` จะห้ามสร้างคำขอใหม่ซ้ำสำหรับอุปกรณ์เดียวกัน
- สำหรับ repair ticket ใช้ enum สถานะ `pending`, `in_progress`, `completed`, `failed` โดย `completed` ทำให้อุปกรณ์กลับเป็น `in_stock` และ `failed` ทำให้อุปกรณ์เป็น `decommissioned`
- Overdue detection สำหรับ MVP จะคำนวณเป็น derived status จากคำขอที่เป็น temporary borrow และอยู่ในสถานะ `approved` โดยถ้า current_date > return_date จะถือว่าเป็น `overdue` เมื่อเรียก query/dashboard/report; ไม่ต้องสร้าง background job ใน MVP
- สำหรับ import endpoint ให้ใช้ไฟล์ CSV ใน MVP พร้อม header row และกฎ validation/duplicate ที่ต้องรายงานผลสรุปเป็น `total`, `success`, `failed` ใน response
- ค่า "ระดับขั้นต่ำ" (minimum quantity) ต่ออุปกรณ์แต่ละประเภทใน MVP นี้เป็นค่า hardcode/seed ไว้ในฐานข้อมูลตายตัว ไม่มี endpoint หรือ UI สำหรับแก้ไขค่านี้ — Admin/System Admin ที่ต้องการเปลี่ยนค่าต้องแก้ผ่าน migration/seed script โดยตรง (deferred เป็น backlog หลัง MVP)

## Acceptance Criteria
AC-1: Admin/System Admin เพิ่มอุปกรณ์ได้
POST อุปกรณ์ใหม่ (asset_name, asset_serial_no, type_id, ...) → ตอบ 201 พร้อมข้อมูล + id
อุปกรณ์ใหม่ตั้งสถานะเริ่มต้นเป็น "ในคลัง" เสมอ (บังคับใน service layer ไม่ให้ client กำหนดสถานะเองตอนสร้าง)
asset_name ว่างเปล่า → ตอบ 422
asset_name ซ้ำกับที่มีอยู่แล้ว → ตอบ 409 ASSET_NAME_DUPLICATED
asset_serial_no ซ้ำกับที่มีอยู่แล้ว → ตอบ 409 ASSET_SERIAL_DUPLICATED

AC-2: ทุก Role ดูรายการอุปกรณ์ได้ (สิทธิ์แก้ไขต่างกัน)
GET รายการ → ได้อุปกรณ์ทุกชิ้น พร้อมสถานะปัจจุบันและผู้ถือครอง (ถ้ามี)
Role "User" เรียก PUT/DELETE บนอุปกรณ์ → ตอบ 403 FORBIDDEN

AC-2b: ทุก Role ดูรายการคำขอเบิกได้
GET รายการ → ได้คำขอเบิกทุกรายการของทุกคน (ทุก role เห็นชุดเดียวกัน ไม่ scope ตามเจ้าของ) พร้อม filter ผ่าน query param `status`, `equipment_id`, `borrower_name`
GET รายการเดี่ยว by id → 200 พร้อมรายละเอียด | 404 ถ้าไม่พบ
สิทธิ์แก้ไข/อัปเดตยังคงจำกัดตาม role และเจ้าของ (ดู AC-5, AC-5b, AC-6)

AC-2c: ทุก Role ดูรายการใบแจ้งซ่อมได้
GET รายการ → ได้ repair ticket ทุกรายการของทุกคน (ทุก role เห็นชุดเดียวกัน ไม่ scope ตามเจ้าของ) พร้อม filter ผ่าน query param `status`, `equipment_id`
GET รายการเดี่ยว by id → 200 พร้อมรายละเอียด | 404 ถ้าไม่พบ
สิทธิ์อัปเดตสถานะจำกัดเฉพาะ Admin/System Admin (ดู AC-8)

AC-3: User สร้างคำขอเบิกอุปกรณ์ที่ว่างได้ (เฉพาะ role User เท่านั้นที่เรียก endpoint นี้ได้)
POST คำขอเบิก (equipment_id, borrow_type, return_date เมื่อ temporary, borrower_name) → 201 — created_by_user_id ตั้งจาก authenticated context เสมอ
borrower_name ว่างเปล่า → 422
เบิกอุปกรณ์ที่สถานะไม่ใช่ "ในคลัง" → 409 EQUIPMENT_NOT_AVAILABLE
(ตัด "borrower_id ต้องเป็น user ที่มีอยู่จริง" และ 404 BORROWER_NOT_FOUND ออกทั้งคู่ เพราะไม่ใช่ FK อีกต่อไป)

AC-4: ห้ามมีคำขอเบิกซ้อนสำหรับอุปกรณ์ชิ้นเดียวกัน (กฎเหล็ก)
อุปกรณ์ที่มีคำขอเบิกสถานะ "รออนุมัติ" หรือ "อนุมัติ" อยู่แล้ว → ยื่นคำขอใหม่ซ้ำ ตอบ 409 EQUIPMENT_ALREADY_REQUESTED
ต้องกันได้แม้มีคำขอเบิกยิงเข้ามาพร้อมกันเป๊ะ (บังคับด้วย Partial Unique Index ที่ database ไม่ใช่แค่เช็คในโค้ด Go)

AC-5: Admin/System Admin อนุมัติคำขอเบิก
PUT อนุมัติ → ตอบ 200, คำขอเปลี่ยนเป็น "อนุมัติ", อุปกรณ์เปลี่ยนเป็น "กำลังใช้งาน" พร้อมแสดงชื่อผู้ใช้งาน
อนุมัติคำขอที่ผ่านการอนุมัติไปแล้ว → ตอบ 409 BORROW_REQUEST_ALREADY_APPROVED
Role "User" เรียก endpoint นี้ → ตอบ 403 FORBIDDEN

AC-5b: Admin/System Admin ปฏิเสธคำขอเบิก
PUT ปฏิเสธ → ตอบ 200, คำขอเปลี่ยนเป็น "ไม่อนุมัติ", asset_state ไม่เปลี่ยน (ยังเป็น "ในคลัง" อยู่แล้วตลอดช่วงรออนุมัติ)
ปฏิเสธคำขอที่ไม่ได้อยู่สถานะ "รออนุมัติ" → ตอบ 409 BORROW_REQUEST_NOT_PENDING
Role "User" เรียก endpoint นี้ → ตอบ 403 FORBIDDEN

AC-6: User แจ้งคืนอุปกรณ์
POST /api/v1/borrow-requests/{id}/return → ตอบ 200, คำขอเปลี่ยนเป็น "รอตรวจรับคืน"
ใช้ได้จากสถานะ "อนุมัติ" และ "เกินกำหนดคืน" เท่านั้น
User แจ้งคืนได้เฉพาะคำขอของตัวเอง; หาก User แจ้งคืนคำขอของคนอื่น → ตอบ 403 FORBIDDEN
แจ้งคืนคำขอที่ไม่ได้อยู่สถานะที่คืนได้ → ตอบ 409 BORROW_REQUEST_NOT_RETURNABLE
equipment.status ไม่เปลี่ยน (ยังเป็น "กำลังใช้งาน" จนกว่า Admin จะยืนยันรับคืน)

AC-6b: Admin/System Admin ยืนยันรับคืน
PUT /api/v1/borrow-requests/{id}/confirm-return → ตอบ 200, คำขอเปลี่ยนเป็น "คืนแล้ว", อุปกรณ์กลับเป็น "ในคลัง"
ยืนยันรับคืนคำขอที่ไม่ได้อยู่สถานะ "รอตรวจรับคืน" → ตอบ 409 BORROW_REQUEST_NOT_PENDING_RETURN
Role "User" เรียก endpoint นี้ → ตอบ 403 FORBIDDEN

AC-6c: ระบบตรวจจับเบิกชั่วคราวเกินกำหนด
คำขอที่ borrow_type = ชั่วคราว และ status = "อนุมัติ" และ current_date > return_date → เปลี่ยนเป็น "เกินกำหนดคืน" อัตโนมัติ (batch job หรือคำนวณตอน query ก็ได้สำหรับ MVP)
แสดงจำนวนนี้แยกใน Dashboard/รายงานได้

AC-7: แจ้งซ่อมอุปกรณ์
POST แจ้งซ่อม (equipment_id, issue_description, urgency) → ตอบ 201, สร้าง repair ticket สถานะ "รอตรวจสอบ", อุปกรณ์เปลี่ยนเป็น "เสียหาย"
แจ้งซ่อมอุปกรณ์ที่ไม่มีอยู่จริง → ตอบ 404 EQUIPMENT_NOT_FOUND
แจ้งซ่อมอุปกรณ์ที่สถานะเป็น "เลิกใช้งาน" อยู่แล้ว → ตอบ 409 EQUIPMENT_DECOMMISSIONED

AC-8: Admin/System Admin อัปเดตสถานะซ่อมจนเสร็จ
PUT อัปเดตสถานะ repair ticket → ตอบ 200
ส่งสถานะที่ไม่อยู่ใน enum (pending/in_progress/completed/failed) → ตอบ 422 INVALID_STATUS
เมื่อสถานะเป็น "ซ่อมเสร็จ" → อุปกรณ์กลับเป็น "ในคลัง" อัตโนมัติ
เมื่อสถานะเป็น "ซ่อมไม่ได้" → อุปกรณ์เปลี่ยนเป็น "เลิกใช้งาน" อัตโนมัติ (terminal, ไม่นับในคลังอีก)
อัปเดตสถานะของ ticket ที่อยู่ terminal state (ซ่อมเสร็จ/ซ่อมไม่ได้) ไปแล้ว → ตอบ 409 REPAIR_TICKET_ALREADY_CLOSED

AC-9: ทุก action ต้องบันทึกลง activity_logs
เบิก/ปฏิเสธ/คืน/อนุมัติ/แจ้งซ่อม/อัปเดตซ่อม/นำเข้า/เพิ่ม-ลบ-แก้ไขอุปกรณ์ ทุกอย่างต้อง insert แถวใหม่ใน activity_logs พร้อม user_id ผู้ทำ
activity_logs ต้องเก็บอย่างน้อย user_id, action, resource_type, resource_id, metadata, created_at
ตรวจสอบได้ผ่าน GET /api/v1/activity-logs (เรียงจากล่าสุด)

AC-10: Role "User" ห้ามแก้ไขอุปกรณ์โดยตรง
PUT/DELETE /api/v1/equipment/{id} จาก role "User" → ตอบ 403 FORBIDDEN เสมอ ไม่ว่าอุปกรณ์จะอยู่สถานะไหน

## Error format (ทั้งระบบ)
{"error": {"code": "MACHINE_READABLE_CODE", "message": "human readable"}}

## API Contract
GET  /health                                     → 200 {"status":"ok"}

GET  /api/v1/equipment                           → 200 [{id, asset_name, asset_serial_no, type_id, asset_state, ...}]
POST /api/v1/equipment {asset_name, asset_serial_no, type_id, ...}
→ 201 | 422 | 409 ASSET_NAME_DUPLICATED | 409 ASSET_SERIAL_DUPLICATED   (Admin/System Admin เท่านั้น)
PUT  /api/v1/equipment/{id}                      → 200 | 404 | 403 FORBIDDEN                 (Admin/System Admin เท่านั้น)
DELETE /api/v1/equipment/{id}                    → 200 | 404 | 403 FORBIDDEN                 (Admin/System Admin เท่านั้น)

POST /api/v1/import (multipart file) → 201 {total, success, failed} (Admin/System Admin เท่านั้น)

GET  /api/v1/borrow-requests?status=&equipment_id=&borrower_name=  → 200 [{id, equipment_id, created_by_user_id, borrower_name, borrow_type, return_date, status, approved_by, returned_at, ...}]
GET  /api/v1/borrow-requests/{id}                → 200 | 404
POST /api/v1/borrow-requests {equipment_id, borrow_type, return_date, borrower_name}
→ 201 | 404 EQUIPMENT_NOT_FOUND
| 409 EQUIPMENT_NOT_AVAILABLE
| 409 EQUIPMENT_ALREADY_REQUESTED
| 422 INVALID_BORROW_TYPE
| 422 (missing return_date for temporary borrow)
| 422 (missing borrower_name)
| 403 FORBIDDEN   ← (Admin/System Admin เรียก endpoint นี้)
(role "User" เท่านั้น)
PUT  /api/v1/borrow-requests/{id}/approve        → 200 | 404 | 409 BORROW_REQUEST_ALREADY_APPROVED  (Admin/System Admin เท่านั้น)
PUT  /api/v1/borrow-requests/{id}/reject         → 200 | 404 | 409 BORROW_REQUEST_NOT_PENDING        (Admin/System Admin เท่านั้น)
POST /api/v1/borrow-requests/{id}/return         → 200 | 404 | 409 BORROW_REQUEST_NOT_RETURNABLE | 403 FORBIDDEN
PUT /api/v1/borrow-requests/{id}/confirm-return  → 200 | 404 | 409 BORROW_REQUEST_NOT_PENDING_RETURN (Admin/System Admin เท่านั้น)


GET  /api/v1/repair-tickets?status=&equipment_id=        → 200 [{id, equipment_id, reporter_id, issue_description, urgency, status, created_at, updated_at, ...}]
GET  /api/v1/repair-tickets/{id}                 → 200 | 404
POST /api/v1/repair-tickets {equipment_id, issue_description, urgency}
→ 201 | 404 EQUIPMENT_NOT_FOUND | 409 EQUIPMENT_DECOMMISSIONED
PUT  /api/v1/repair-tickets/{id}/status {status} → 200 | 404 | 422 INVALID_STATUS | 409 REPAIR_TICKET_ALREADY_CLOSED  (Admin/System Admin เท่านั้น)

GET  /api/v1/dashboard                           → 200 (KPI ตาม role ที่login)
GET  /api/v1/reports?type=stock|borrow|return|repair&from=&to= → 200   (Admin/System Admin เท่านั้น)
GET  /api/v1/activity-logs                       → 200 [{id, user_id, action, resource_type, resource_id, metadata, created_at}]

## database schema
CREATE TABLE IF NOT EXISTS users (
  id       SERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE CHECK (length(trim(username)) > 0),
  role     TEXT NOT NULL CHECK (role IN ('system_admin', 'admin', 'user'))
);

CREATE TABLE IF NOT EXISTS equipment_types (
  id           SERIAL PRIMARY KEY,
  name         TEXT NOT NULL UNIQUE CHECK (length(trim(name)) > 0),
  min_quantity INTEGER NOT NULL DEFAULT 0 CHECK (min_quantity >= 0)
);

CREATE TABLE equipment (
    id SERIAL PRIMARY KEY,
    type_id INTEGER NOT NULL REFERENCES equipment_types(id),
    product_name VARCHAR(255),
    asset_name VARCHAR(255) UNIQUE NOT NULL CHECK (length(trim(asset_name)) > 0),
    asset_tag VARCHAR(255),
    asset_serial_no VARCHAR(100) UNIQUE,
    bar_code VARCHAR(255),
    vendor_name VARCHAR(255),
    location VARCHAR(255),
    assigned_to_department VARCHAR(255),
    site VARCHAR(255),
    asset_no VARCHAR(255),
    budget VARCHAR(10),
    remark TEXT,
    req_no TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'ในคลัง'
        CHECK (status IN ('กำลังใช้งาน', 'ในคลัง', 'เสียหาย', 'เลิกใช้งาน')),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
-- current_holder ไม่ใช่ column: derived จาก borrow_records ที่ status='อนุมัติ' ตอน query (ดู T-08)

CREATE TABLE borrow_records (
    id SERIAL PRIMARY KEY,
    equipment_id INTEGER NOT NULL REFERENCES equipment(id),
    created_by_user_id INTEGER NOT NULL REFERENCES users(id),
    borrower_name VARCHAR(255) NOT NULL CHECK (length(trim(borrower_name)) > 0),
    approved_by INTEGER REFERENCES users(id),
    borrow_type VARCHAR(20) NOT NULL
        CHECK (borrow_type IN ('เบิกถาวร', 'เบิกชั่วคราว')),
    return_date DATE,
    reason TEXT,
    remark VARCHAR(255),
    status VARCHAR(30) NOT NULL DEFAULT 'รออนุมัติ'
        CHECK (status IN ('รออนุมัติ', 'อนุมัติ', 'ไม่อนุมัติ',
                           'รอตรวจรับคืน', 'คืนแล้ว', 'เกินกำหนดคืน')),
    requested_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMPTZ,
    returned_at TIMESTAMPTZ,
    CONSTRAINT chk_return_date_matches_type CHECK (
        (borrow_type = 'เบิกชั่วคราว' AND return_date IS NOT NULL)
        OR (borrow_type = 'เบิกถาวร' AND return_date IS NULL)
    )
);

CREATE UNIQUE INDEX one_active_borrow_per_equipment
    ON borrow_records (equipment_id)
    WHERE status IN ('รออนุมัติ', 'อนุมัติ', 'รอตรวจรับคืน', 'เกินกำหนดคืน');

CREATE TABLE repair_records (
    id SERIAL PRIMARY KEY,
    equipment_id INTEGER NOT NULL REFERENCES equipment(id),
    reporter_id INTEGER NOT NULL REFERENCES users(id),
    issue TEXT NOT NULL,
    remark TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'ส่งซ่อม'
        CHECK (status IN ('ส่งซ่อม', 'กำลังซ่อม', 'ซ่อมเสร็จ', 'ซ่อมไม่ได้')),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS activity_logs (
  id            SERIAL PRIMARY KEY,
  user_id       INTEGER NOT NULL REFERENCES users(id),
  action        TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id   INTEGER NOT NULL,
  metadata      JSONB,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);