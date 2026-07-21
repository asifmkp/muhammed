# NTBF Platform — Comprehensive Engineering & Business Architecture Audit

| | |
|---|---|
| **Repository** | `asifmkp/ntbf-platform` @ `61a9e7c` (main) |
| **Audit date** | 2026-07-21 |
| **Method** | 7 parallel read-only auditors (architecture/data-flow, database/auth, Zoho/WhatsApp, AI/OCR, infrastructure/security, code quality/debt, quantitative census + workflow tracing), findings cross-verified against code with `file:line` evidence |
| **Mode** | Strictly read-only — no application code was modified |
| **Convention** | Every issue carries: current implementation → concern → recommended solution → effort (Quick <1d / Small 1–3d / Medium 1–2wk / Large >2wk) → business impact → priority (P0–P3) |

> ⚠️ **Redaction note:** the platform repo contains real hardcoded credentials. Their values are deliberately **not** reproduced in this document — only their locations. See finding SEC-01.

---

## 1. Executive Summary

NTBF Platform is the operations system of National Trading of Beverage & Foodstuff LLC (Ajman, UAE): a customer ordering PWA, a role-locked staff field app, a Claude-powered WhatsApp ordering bot, and Zoho Books (org `928751913`) as the accounting system of record. It runs as **one NestJS container on Render** with a 1 GB persistent disk.

**The five facts that matter most:**

1. **Two parallel backends coexist.** "System B" — a complete, well-designed Prisma/PostgreSQL ERP (46 models, ~80 endpoints) — is **dead in production** (no database is provisioned). The **live** business runs on "System A": JSON files on a single 1 GB disk. `README.md` and `IMPLEMENTATION_STATUS.md` describe the dead system as the product; the live system is documented only in `CLAUDE.md` and `ai/`.

2. **The entire business's live data sits on one unbacked disk, behind code that swallows write errors.** Every order, receipt, payment, advance, staff account, and the audit chain live in `/var/data` JSON files. There is **no backup, no restore procedure**, and every store's `save()` silently ignores failures (`catch { /* ignore */ }`) — a full disk means the API reports success while persisting nothing. This is the single highest expected-loss item in the audit (the project's own RISK-001 agrees).

3. **Credentials are exposed and rotation is overdue — and the repo is public.** Real staff usernames and passwords are hardcoded in source (`staff-auth.module.ts:56-61`) in a **publicly visible GitHub repository**; the Zoho OAuth client secret + refresh token and the Anthropic API key were pasted into chats ("burned") and are flagged OVERDUE for rotation in `CLAUDE.md:49-52`. `JWT_SECRET` falls back to the literal string `'dev-secret'` in 11 files if the env var is ever unset.

4. **The order-to-cash loop does not close.** The live pipeline goes PLACED→…→DELIVERED with role-gated transitions (good), but: no invoice is ever generated, nothing reconciles finance receipts against orders, the driver-entered cash-on-delivery amount has **no upper bound**, no stock is ever decremented, and there is **no app→Zoho sync** — live transactions accumulate un-booked in accounting (the project's own "biggest gap", TASK-012).

5. **The AI governance layer (`ai/`) is exceptional** — evidence-graded fact registers, append-only decision log, task queue, traceability records. Most findings in this audit corroborate risks the project already recorded. The gap is **enforcement**: no CI exists (TESTING.md links a workflow file that doesn't exist), merge-to-main auto-deploys to production, and only ~21 unit tests exist — none on the money paths.

**Overall verdict:** the live system is small, coherent, and honestly self-documented, with several genuinely good controls (server-side re-pricing, Zoho write-lock + org allow-list, dual-control approvals, hash-chained audit log, idempotent WhatsApp ingest). The dominant risks are **operational, not algorithmic**: one unbacked disk, exposed credentials, fail-open configuration defaults, no CI, and an unreconciled money loop. Nearly all P0 fixes are Quick/Small — roughly one focused week of work removes the majority of the existential risk.

---

## 2. Quantitative Metrics (measured from code, not docs)

| Metric | Count | Notes |
|---|---|---|
| NestJS modules | **31** | `*.module.ts` = `@Module(` decorators = 31 |
| Controllers | **29** | 11 dedicated files + 18 declared inline in module files |
| API endpoints | **197** | All under `/api` prefix; full inventory captured during audit |
| Services | **33** service classes (15 dedicated files); 52 `@Injectable` total | |
| Prisma models / enums | **46 / 42** | `schema.prisma`, 1,002 lines — unused in production |
| Scheduled jobs (server) | **0** | No `@Cron`/`node-cron`/`setInterval` anywhere in backend; only 3 client-side polls (6 s state sync, 30 s badges, 20 s outbox) |
| Runtime AI assistants | **3** | Bills vision OCR; "Muhammed" staff assistant (24 read-only tools); legacy copilot agent (28 tool schemas, orphaned) — plus the out-of-repo WhatsApp bot |
| Environment variables | **30 distinct** | 5 undocumented in `.env.example`: `STATE_DIR`, `STATIC_DIR`, `ZOHO_WRITES_ENABLED`, `AUDIT_EXPORT_ENABLED`, `AUDIT_SUPABASE_URL/KEY` |
| Docker services | compose: **2** (app + postgres); Render: **1** web service + 1 GB disk | No database in `render.yaml` despite DEPLOY.md claiming one |
| Lines of code | backend/src **8,260** (81 TS files); apps **≈6,358** (mobile-app 4,059 — one file `app.js` is 3,104) | |
| Tests | **3 test files, ~21 tests** | Zero coverage on finance, orders, auth/guards |
| Backend framework | **NestJS 10.4** (verified from package.json + bootstrap, not assumed) | |

---

## 3. Architecture Overview

### 3.1 High-level architecture (diagram)

```
                                ┌──────────────────────────────────────────┐
                                │   RENDER (single Docker container,       │
                                │   starter plan, 1 instance)              │
 Customer phone                 │                                          │
 ┌────────────┐   Meta Cloud    │  ┌────────────────────────────────────┐  │
 │  WhatsApp  │──► API (360-    │  │        NestJS backend (/api)       │  │
 └────────────┘   dialog)       │  │                                    │  │
       │                        │  │  SYSTEM A (LIVE)     SYSTEM B      │  │
       ▼                        │  │  ─────────────────   (DORMANT)     │  │
 ┌──────────────────┐  x-ingest │  │  customer-portal     auth          │  │
 │ Supabase Edge Fn │──token───►│  │  staff-auth          orders        │  │
 │ whatsapp-webhook │           │  │  finance             catalog       │  │
 │ (SOURCE NOT IN   │           │  │  rashid (expenses)   payments      │  │
 │  THIS REPO)      │           │  │  bills / documents   inventory     │  │
 └──────────────────┘           │  │  muhammed (AI)       procurement   │  │
       │                        │  │  appstate / audit    hr / sales    │  │
       ▼                        │  │  dashboard / zoho    delivery      │  │
 Supabase Postgres              │  │  agent (orphaned)    accounting…   │  │
 (wa_messages, wa_orders)       │  └───────┬──────────────────┬─────────┘  │
                                │          │                  │            │
 Staff phones                   │          ▼                  ▼            │
 ┌──────────────────┐           │  /var/data JSON files   PostgreSQL      │
 │ Staff PWA        │──JWT─────►│  (1 GB disk — THE       (NOT PROVISIONED│
 │ apps/mobile-app  │           │  system of record,      IN PRODUCTION)  │
 └──────────────────┘           │  NO BACKUP)                             │
 ┌──────────────────┐           │                                          │
 │ Customer PWA     │──JWT─────►│  ServeStaticModule also serves all      │
 │ apps/order       │           │  front-ends from /app/apps               │
 └──────────────────┘           └───────────────┬──────────────────────────┘
                                                │
                              ┌─────────────────┼──────────────────┐
                              ▼                 ▼                  ▼
                        Zoho Books        Anthropic API      (Groq — voice,
                        org 928751913     (Claude, raw        out-of-repo
                        writes env-locked  fetch, no SDK)     bot only)
```

### 3.2 Component inventory

- **`backend/`** — NestJS 10, serves API + all static apps. Two subsystems (§1 fact 1).
- **`apps/mobile-app/`** — the real staff app (orders queue, finance, expenses, bills OCR, Muhammed chat, team admin). One 3,104-line `app.js` covering all 8 roles.
- **`apps/order/`** — customer ordering PWA (register/login/order/track).
- **`apps/role-dashboards/`, `apps/admin-dashboard/`** — demo-era Zoho snapshot dashboards (fictional persona data, snapshot fallback). Vestigial.
- **`apps/native/`** — minimal Expo shell (281 LOC), early phase.
- **`ai/`** — 15-file multi-agent governance protocol (task queue, decisions, facts, risks, traceability). Not runtime code.
- **WhatsApp bot** — Supabase Edge Function (v41), **source outside this repo, no version control** (finding WA-01).

---

## 4. Module Dependency Map

```
app.module ──► all 31 modules          main ──► app.module

SYSTEM A (live)                          SYSTEM B (dormant, all ──► prisma)
staff-auth (identity hub, leaf)          auth, catalog, customers, orders,
customer-portal ──► staff-auth           payments, inventory, procurement,
rashid ──► staff-auth                    hr, sales, delivery, accounting,
suggestions ──► staff-auth               support, notifications(stub), prisma
appstate ──► common
finance ──► staff-auth, customer-portal, rashid
attention ──► staff-auth, customer-portal, finance, rashid, suggestions
admin/clear-test-data ──► staff-auth, appstate, customer-portal, rashid, finance, suggestions
admin/july-backfill ──► staff-auth, customer-portal, rashid, finance
audit ──► staff-auth  (global APP_INTERCEPTOR)

AI / ZOHO
zoho (leaf) ◄── bills, documents, dashboard
ai/AnthropicService ◄── bills, agent, muhammed   (3 separate instances — no shared module)
muhammed ──► ai, appstate, staff-auth
```

**Circular dependencies: none.** Hotspots: `finance.module.ts` (866 lines, all DTOs/stores/services/controllers in one file), `customer-portal.module.ts` (476 lines — the de facto orders database), `staff-auth` (imported by 8 modules). Duplicated infrastructure: the JWT config factory (with its `'dev-secret'` fallback) is copy-pasted in 7+ modules; the atomic JSON-store pattern is re-implemented 6×; `AnthropicService` instantiated 3×. Dead code: all 14 System-B modules, the `agent` copilot (its browser executor `copilot.js` no longer exists), notifications FCM stub, demo dashboards, `attachToRecord` stub.

---

## 5. Data Flow Diagrams

### 5.1 Request lifecycle (System A route)

```
HTTPS request
  ─► Render edge ─► NestJS (single process)
  ─► Global ValidationPipe (whitelist + forbidNonWhitelisted)     [good]
  ─► Global JwtAuthGuard … BYPASSED via @Public() on every System-A route
  ─► Per-route StaffAuthGuard / CustomerAuthGuard / ApiGateGuard  [must be remembered per route — fail-open pattern, SEC-08]
  ─► Controller ─► Service ─► JSON store (synchronous whole-file write, atomic tmp+rename)
  ─► Global AuditInterceptor (hash-chained audit-log.json, fail-open)
  ─► Response
```

### 5.2 Authentication flow

```
STAFF:    POST /api/staff/login {username,password}
            ─► bcrypt compare vs data/staff.json
            ─► JWT {sub, typ:'staff', name, roles}  — 30-day expiry, no refresh
            ─► stored in browser localStorage
            ─► sent as Authorization: Bearer; StaffAuthGuard verifies typ==='staff'
            ─► role checks ad-hoc in services (assertAdmin / hasRole / isFinance)

CUSTOMER: POST /api/portal/login ─► JWT {sub, typ:'customer'} (same JWT_SECRET)

SHARED-SECRET GATES:
  x-api-key == PUBLIC_API_TOKEN  ─► ApiGateGuard (agent/dashboard/appstate/documents/bills)
      └── if PUBLIC_API_TOKEN unset ⇒ GATE IS OPEN (fail-open, SEC-03)
  x-ingest-token == WHATSAPP_INGEST_TOKEN ─► ingest + muhammed/wa (plain string compare, no HMAC)

DORMANT (System B): POST /api/auth/register|login ─► Prisma User + global guards
      └── register is PUBLIC and accepts client-supplied role/department/accessLevel
          up to SUPER_ADMIN (SEC-04)
```

### 5.3 Zoho integration flow

```
READS (live): dashboard ─► ZohoService.get() ─► OAuth refresh-token grant
   (in-memory token cache, 60s buffer, no mutex/retry)
   ─► items/invoices/contacts/SOs/POs … single page, per_page=200, NEVER paginates
   (org has 1,490 items ⇒ KPIs/matching see ~13%)  [ZH-02]

WRITES (env-locked): bills/documents ─► ZohoService.post()
   ─► guard 1: ZOHO_WRITES_ENABLED !== 'true' ⇒ refuse (currently OFF in prod)
   ─► guard 2: org != 928751913 ⇒ refuse (hard-coded allow-list)   [good]
   ─► POST bills | expenses | purchaseorders | contacts(vendor)
   ─► NO idempotency, NO retry, NO duplicate check                  [ZH-01]
   attachToRecord() = stub returning fake {attached:true}           [ZH-06]

NO app→Zoho sync exists. Live transactions accumulate un-booked.   [ZH-09 / P1-01]
```

### 5.4 WhatsApp message flow

```
Customer msg ─► Meta Cloud API (360dialog) ─► Supabase Edge Fn (OUT OF REPO, v41)
   ─► Claude conversation, catalog match, save_order ─► wa_orders (Supabase)
   ─► POST /api/portal/orders/ingest  [x-ingest-token; IngestGuard fail-closed]
        ─► idempotent by (source:'whatsapp', external_ref)          [good]
        ─► free-text lines ─► server-side catalog matcher, server re-pricing
        ─► unmatched/unpriced/unknown-phone ⇒ needsReview (blocks CONFIRMED)  [good]
        ─► data/customers.json  status=PLACED
   ─► bot stores platform order id
Staff Q&A: bot ─► POST /api/muhammed/wa [same shared secret]
   └── bot may assert name/roles for unknown phones — incl. admin   [AI-03]
Bot→backend retry behavior: UNKNOWN (bot code unauditable)          [WA-03]
Outbound sending, templates, media, 24h-window: all in the out-of-repo bot.
```

### 5.5 AI orchestration flow (Muhammed)

```
/api/muhammed/ask [StaffAuthGuard]  or  /wa [ingest token]
  ─► MuhammedService: role-filtered read-only toolset (24 tools; mapping enforced
     in code, not prompt)  [good]
  ─► Anthropic tool loop, MAX 15 calls / 6 rounds, max_tokens 800
  ─► tools read appstate.json (the field-app sync blob) + StaffStore
        └── appstate is writable via weakly-gated PUT ⇒ prompt-injection path [AI-04]
  ─► answer logged to muhammed-log.json (cap 5,000, indefinite retention)
Cost controls: NONE (no token accounting, budget, or per-user quota) [AI-02]
```

### 5.6 Order lifecycle (live pipeline — as actually implemented)

```
PLACED ──► CONFIRMED ──► PACKED ──► OUT_FOR_DELIVERY ──► DELIVERED
(customer/  (salesman;   (warehouse) (warehouse|driver)   (driver; records
 WhatsApp)   blocked by                                    collected{amount…}
             needsReview)                                  — NO UPPER BOUND)
   └────────────── CANCELLED (sales/admin; admin any-pair override) ─────────┘

MISSING STAGES (do not exist in the live system):
  ✗ Invoice generation        ✗ Payment↔order reconciliation
  ✗ Stock decrement           ✗ Customer notifications
  ✗ Zoho sales booking        (finance receipts are a separate, manually-linked flow)
```

---

## 6. Database & Storage Analysis

**Storage reality:** production data = JSON files under `/var/data` (Render 1 GB disk): `staff.json`, `customers.json` (all orders), `appstate.json`, `finance-{receipts,payments,transfers}.json`, `expenses.json`, `advances.json`, `suggestions.json`, `muhammed-log.json`, `audit-log.json`, plus binary photo dirs `finance-bills/`, `expense-bills/` (15 MB uploads, never cleaned up). Atomic tmp+rename writes (good) but **all writes swallow errors silently** and there is **no cross-file transaction, no locking, no backup**.

**Prisma schema (dormant, 46 models):** competent design — uuid PKs, `Decimal(14,2)` money, composite uniques (`Inventory`, `Attendance`, `Payroll`), enums throughout. Defects:

| ID | Finding | Effort | Impact | Priority |
|---|---|---|---|---|
| DB-01 | **No migration files exist** — `prisma/migrations/` absent; documented `prisma migrate deploy` no-ops; Dockerfile never migrates | Small | Schema drift; deploy path broken | P1 |
| DB-02 | ~30 "actor" FKs (`approvedById`, `collectedById`…) are unconstrained `String?` — no referential integrity on approvals | Medium | Untrustworthy audit trails | P2 |
| DB-03 | Stock duplicated: `Product.stockQty` vs `Inventory.quantityOnHand`, different writers | Medium | Divergent stock, overselling | P2 |
| DB-04 | Line items as `Json` (PO, GRN, requisition, returns) — unqueryable, unvalidated | Medium | No procurement reporting | P3 |
| DB-05 | No soft-delete; no `updatedAt` on financial rows | Small | Lost history on mutation | P2 |
| DB-06 | Missing indexes on date/status filter columns used by reports | Quick | Full scans at scale | P2 |
| DB-07 | PrismaService boots with DB unreachable (`dbReady=false`) — silent partial availability | Quick | Misconfig invisible | P2 |
| DB-08 | Payroll run: N+1 (~3N queries) **and no transaction** — mid-run failure = partially paid payroll | Small | Payroll correctness | P2 |
| DB-09 | 40+ unbounded `findMany` (no pagination) incl. full-table accounting scans | Small–Medium | Memory/latency growth, DoS | P1 |
| DB-10 | Live money data on non-transactional JSON: read-modify-write races, last-write-wins, single-instance lock-in | Medium–Large | Lost/corrupt financial writes | P1 (design) / P2 (migrate) |
| DB-11 | Bill/expense photos never deleted on clear/archive → 1 GB disk exhaustion → silent write loss | Small | Disk-full cascade | P1 |

---

## 7. Authentication & Authorization Flow

Three auth mechanisms (staff JWT, customer JWT, shared-secret gates) + a dormant fourth (Prisma JWT), all signing with one `JWT_SECRET`. Flow diagrams in §5.2. Findings consolidated in the Security Assessment (§14) — headline items: hardcoded staff passwords (SEC-01), `'dev-secret'` fallback ×11 (SEC-02), fail-open API gate (SEC-03), public SUPER_ADMIN self-registration on the dormant path (SEC-04), the `@Public()`-then-re-guard fail-open pattern (SEC-08), 4-char password minimums and 30-day non-refreshable tokens on the money system (SEC-09), tokens + shared API key in `localStorage` (SEC-10).

---

## 8. Zoho Integration Audit

**Map:** single thin client `zoho.service.ts` (193 lines, raw fetch). Reads: items/invoices/contacts/SOs/POs for dashboard KPIs + bill matching. Writes (env-locked OFF): bills, expenses, POs, vendor contacts. OAuth refresh-token grant with in-memory cache. **No local↔Zoho entity mapping is persisted anywhere.**

| ID | Issue | Current implementation | Concern | Solution | Effort | Impact | Priority |
|---|---|---|---|---|---|---|---|
| ZH-01 | No write idempotency | `bills.service.ts:100-135`, `documents.module.ts:94-128` — no duplicate lookup, no idempotency key, no posted-log | Double-tap/retry/5xx-after-success posts bills & vendors **twice**; corrupts AP/VAT in the ledger of record | Pre-create lookup by `bill_number`; persist `{localRef→zoho_id}`; reuse the platform's `clientRef` convention | Small | Launch blocker for the write path | **P1** (P0 before enabling writes) |
| ZH-02 | Pagination capped at 200 | `zoho.service.ts:113-131,188` single page; org has 1,490 items | KPIs/stock/counts ~13% of reality; bill-line matching misses 87% of catalog | Paged loop on `has_more_page`; use `page_context.total` for counts | Quick | Wrong owner KPIs; bad OCR matches | **P1** |
| ZH-03 | No retry/backoff/429 handling; public dashboard fires 6+ uncached Zoho calls per view | `zoho.service.ts:97-158` | Transient blips → user-facing 503s; quota exhaustion takes down integration | Bounded retry + jitter; honor `Retry-After`; 30–60 s dashboard cache | Small | Reliability + quota headroom for future sync | P1 |
| ZH-04 | Wrong org + `.ae` DC shipped in repo defaults | `render.yaml:24-29`, `.env.example:17-27`, code defaults `.ae` — vs the hard rule (only `928751913` on `.com`) | Env rebuild from repo points at wrong org; reads are **not** org-guarded | Fix defaults; make org mandatory; extend org check to reads | Quick | Config landmine (project's RISK-007/TASK-023) | **P0** |
| ZH-05 | Zoho OAuth creds burned, rotation OVERDUE | `CLAUDE.md:50` | Long-lived credential to the full ledger exposed via chat transcripts | Regenerate Self Client into Render; revoke old token | Quick (owner) | Open credential exposure on accounting | **P0** |
| ZH-06 | Attachment success is fabricated | `attachToRecord` stub returns `{attached:true}` (`zoho.service.ts:176-180`) | Users believe bill photos are attached in Zoho; audit trail silently absent | Implement multipart attach or return honest `false` | Small | VAT/audit document chain | P1 |
| ZH-07 | Zoho write endpoints lack staff auth | `/api/documents/post`, `/zoho-test-po` behind ApiGateGuard only; guard never attaches identity → audit actor `anonymous` | Once writes enabled: unattributed ledger writes by any key/JWT holder incl. drivers | `StaffAuthGuard` + role check; attach identity for audit | Quick | Prerequisite to enabling writes | **P1** |
| ZH-08 | `revenueMtd` is all-time invoice total | No date filter (`zoho.service.ts:118-122`) | Mislabeled owner KPI | Add month date range or rename | Quick | Decision accuracy | P2 |
| ZH-09 | **No app→Zoho sync** | Confirmed absent (I-08, TASK-012) | Books drift daily; un-booked revenue; VAT exposure; compounded by no backup | Build daily sync (idempotent, `july-import` excluded); **interim: nightly backup** | Large (sync) / Quick (backup) | Biggest financial-integrity gap | **P1** (backup **P0**) |

---

## 9. WhatsApp Integration Audit

**Map:** Meta WhatsApp Cloud API via 360dialog; webhook handled by a **Supabase Edge Function whose source is not in this repository**. In-repo surfaces: the frozen idempotent order-ingest endpoint (good) and the Muhammed staff-assistant endpoint, both guarded by one shared static secret.

| ID | Issue | Current implementation | Concern | Solution | Effort | Impact | Priority |
|---|---|---|---|---|---|---|---|
| WA-01 | **Production bot has no version control** | Deployed edge function "is the truth"; only stale local copy; no VCS documented | Customer-facing front door is un-auditable, unrecoverable; webhook verification/templates/media/24h-window **cannot be audited** | Vendor the deployed source into the repo; deploy from git | Quick–Small | Recoverability of the primary sales channel | **P0** |
| WA-02 | Muhammed dedupe is in-memory, wholesale-cleared | `waSeen` Set, `clear()` past 5,000, empty on restart | WhatsApp redeliveries → duplicate AI replies + duplicate Anthropic spend | Persist seen ids (TTL 24 h) or dedupe on Meta `wamid` in Supabase; LRU not clear() | Quick | Cost + staff confusion | P1 |
| WA-03 | Bot→backend ingest retry behavior UNKNOWN | Repo endpoint is idempotent, but nothing in repo guarantees retries happen | A confirmed customer order can exist only in Supabase and **never reach the staff queue** — silently lost sale | `delivered_to_platform` flag + scheduled retry in bot; admin reconciliation view | Small | Lost revenue path | **P1** |
| WA-04 | Shared-secret hygiene | Token check duplicated; `!==` compare; no rotation story | Drift risk; no safe rotation | Consolidate on IngestGuard; `timingSafeEqual`; dual-token rotation | Quick | Hygiene | P2 |
| WA-05 | Voice notes unverified; template/24h-window handling invisible | "Wired, key verification pending" (TASK-003) | Voice orders may silently fail; template wall unprepared for outbound features | Execute the live voice test; document window behavior once WA-01 lands | Quick | Channel completeness | P2 |
| WA-06 | No HMAC/replay protection on ingest | Plain bearer secret; client-supplied `name`/`roles` in `/muhammed/wa` body | Leaked token ⇒ fake orders + staff impersonation incl. admin reads | HMAC-SHA256 over raw body + timestamp window; role allow-list (never admin) | Small | Order forgery / data exposure | **P1** |

---

## 10. Order Lifecycle & Inventory Synchronization

Traced through code (§5.6). The live pipeline's role-gated transition matrix, needsReview gate, and statusHistory audit are genuinely good. The gaps:

| ID | Gap | Current implementation | Concern | Solution | Effort | Impact | Priority |
|---|---|---|---|---|---|---|---|
| OL-01 | COD amount unbounded | `customer-portal.module.ts:360-364` accepts any `amt >= 0` incl. 0 or 99999 on a 50-AED order | Direct cash-leak/typo exposure on the main money channel; EOD trusts it | Reject `amt > total(+tolerance)`; short-collection requires reason → approval queue | Quick | High — daily cash control | **P0** |
| OL-02 | Order-less receipts unbounded | `finance.module.ts:255-271` — server bill cross-check bypassed by omitting `orderId` | Fabricated/fat-fingered money-in records | Require orderId or billAmount, or force PENDING_APPROVAL | Quick | Med-High | **P0** |
| OL-03 | No invoice stage & no payment↔order reconciliation | DELIVERED ends at a JSON `collected{}` field; receipts separately created, never linked back | No system-enforced "every delivered order got paid"; AR invisible; revenue leakage | Auto-create receivable on DELIVERED; reconcile receipts; unpaid-delivered report | Medium | High | **P1** |
| OL-04 | Live orders never touch stock | Portal prices from static catalog; no stock model in live path | Unbounded overselling; warehouse blind to committed stock | Introduce stock on live path (or route through inventory) with decrement-on-confirm | Medium | High | P1 |
| OL-05 | No undo/correction anywhere | Every terminal state is a dead end; no reversal endpoint | Mistakes become permanent bad data or ad-hoc server-side JSON edits (outside the audit chain) | Time-boxed, admin-approved compensating-record reversal | Medium | High for books accuracy | P1 |
| OL-06 | Admin any-pair override | `:352` — PLACED→DELIVERED in one call, with admin-typed cash | Combined with OL-01: fabricate delivered+collected orders | Restrict to adjacent transitions or require mandatory note + attention alert | Quick-Small | Segregation of duties | P1 |
| OL-07 | Prisma oversell race (dormant) | Stale check-then-decrement, no conditional guard | Negative stock under concurrency if System B ever activates | `updateMany WHERE stockQty >= qty` or row lock | Quick | Latent | P2 |
| OL-08 | Dormant-pipeline dead ends | Delivery rows never created; `Invoice.status`/`Order.paymentStatus` never written; any-status transitions; GRN accept updates **no stock**; transfer double-receive possible | Schema promises states it can't deliver; broken procurement→inventory chain | Fix only if System B is revived — DEC-018 says delete instead | n/a | Informational | P3 |
| OL-09 | Inventory in 4 disconnected places | `Product.stockQty`, `Inventory`, static catalog (no stock), Zoho `stock_on_hand` (read-only); **no sync in any direction** | No single truth for stock anywhere | Decide the source of truth as part of the Zoho-sync design | Design item | High | P1 (design) |
| OL-10 | Catalog price data duplicated ×4, hand-regenerated | `catalog.data.ts` ← `mobile-app/catalog.js`; more copies in order/native | Drift silently mis-prices real orders | Single source served from one endpoint; build-time consistency check | Small | Revenue accuracy | P1 |
| OL-11 | Zero scheduled jobs | No cron anywhere; renewals-due/EOD digests/sync all need a human to poll | Nothing time-driven can exist (incl. backups & sync) | Add `@nestjs/schedule`; first jobs: backup, unpaid-delivered digest | Small | Enabler | P1 |

---

## 11. AI / Agent Architecture Audit

**Operating model:** the `ai/` directory implements a disciplined repository-as-memory multi-agent protocol (task queue with claim-by-commit, append-only decisions DEC-001…019, evidence-graded fact register, traceability records, orchestrator deliberately postponed behind 5 gates). Governance quality is **high**; weaknesses are conventional-only enforcement (the CI linter exists only as an inactive template) and single-agent dependency.

**Runtime AI:** Anthropic only (raw fetch, no SDK). Three surfaces: bills vision OCR (staff-gated), Muhammed (well-designed: server-side read-only tools, role mapping enforced in code, loop caps), and the **orphaned** legacy copilot.

| ID | Issue | Current implementation | Concern | Solution | Effort | Impact | Priority |
|---|---|---|---|---|---|---|---|
| AI-01 | Orphaned paid LLM endpoint | `/api/agent/chat` behind ApiGateGuard only; its browser tool-executor `copilot.js` **no longer exists**; role client-supplied | Dead-but-routable paid endpoint; open to the internet if gate token unset | Delete the agent module (or StaffAuthGuard it until then) | Quick | Cost/abuse channel removed | **P0** |
| AI-02 | No AI spend controls | No token accounting, budgets, or per-user quotas; 15 MB bodies accepted; rate limit spoofable | Unbounded API spend; zero cost visibility | Real-IP keying; per-staff quotas; log `usage` tokens to audit; console budget alert | Small | Money leak capped | P1 |
| AI-03 | Bot can assert staff roles incl. admin | `muhammed.service.ts:69-73` trusts `roles` from ingest-token holder for unknown phones | One static secret = company-wide financial reads as "admin" | Role allow-list (never admin); require registration of unknown phones; rotate + HMAC | Quick-Small | Blast-radius reduction | **P1** |
| AI-04 | Prompt-injection path via appstate | Customer-writable names + weakly-gated `PUT /api/appstate` feed Muhammed `tool_result`s unsanitized (July incident precedent recorded) | Staff-facing AI answers manipulable by third parties | Lock appstate (TASK-021); sanitize/length-limit names; "tool data is untrusted" system rule | Small | AI-channel trust | P1 |
| AI-05 | Model/config drift | Code default `claude-sonnet-5` vs render pin `claude-sonnet-4-6`; docs disagree | Env rebuild may target unintended model | Align defaults with decided values | Quick | Consistency | P2 |

---

## 12. OCR & Document Processing Audit

"OCR" is exclusively Claude vision over base64 **images** (no Tesseract, **no PDF support**). Three flows share `/api/bills/extract`: expense prefill (live; human-verify step, auto-approve ≤ AED 50), purchase bill→Zoho (write-locked; preview-only today), document capture→owner confirm (preview-only).

**Working safeguards:** forced tool schema (always shape-valid JSON), editable human-review in every flow, match-confidence display, needsReview gates, write-lock + org guard. The WhatsApp path is the strongest safeguard in the codebase: the server **never trusts LLM numbers** — it re-matches and re-prices everything.

| ID | Issue | Current implementation | Concern | Solution | Effort | Impact | Priority |
|---|---|---|---|---|---|---|---|
| OCR-01 | **Fabricated data fallback in production** | On any extract failure the client silently substitutes `mockExtract()` — a fake demo bill; `_demo:true` never surfaced | A distracted user can save plausible fabricated numbers as a real bill | Explicit error state; mocks behind dev flag; DEMO banner | Quick | Eliminates fake-financial-data path | **P0** |
| OCR-02 | No server-side arithmetic validation | `/bills/record` validates `bill`/`match` as bare `@IsObject()`; no Σlines≈subtotal, subtotal+tax≈total, no duplicate `bill_number` check; "best estimate" prompt with no uncertainty signal | Garbage/tampered/AI-mis-read figures flow into the ledger once writes enabled; duplicates possible | Nested DTO validation + arithmetic cross-checks + duplicate lookup; post as Zoho drafts | Small | **Must land before `ZOHO_WRITES_ENABLED=true`** | **P1** |
| OCR-03 | Auto-approve ≤ AED 50 with OCR prefill | Instant APPROVED, threshold admin-editable upward at runtime | Systematic small errors accumulate into cash-reconciliation drift | Cap threshold in code; require photo; periodic admin review view | Quick | Bounded convenience | P2 |
| OCR-04 | No timeout on `extractBill`; no PDF support | Unlike ping/createMessage, no AbortController; PDF invoices must be photographed | Hung upstream ties up requests; quality loss on the dominant invoice format | Add 30–60 s abort; accept PDF via document content blocks | Quick / Small | Robustness + accuracy | P2 |
| OCR-05 | Privacy disclosure under-covers AI processing | Privacy page mentions doc-text extraction only; Muhammed chat, WA conversations, Groq voice, indefinite Q&A log retention undisclosed | UAE PDPL-style exposure; customer trust | Update disclosure (Anthropic, Groq, 360dialog, Supabase); add retention window; include log in backups | Quick-Small | Compliance posture | P2 |

---

## 13. Infrastructure & Deployment Audit

**Topology:** one Docker container on Render (starter, single instance) + 1 GB disk. No database in `render.yaml` (docs claim one — false). Merge to `main` auto-deploys ("merge = ship") with **no CI** — `.github/` holds only a PR template; TESTING.md links a nonexistent `ci.yml`. No zero-downtime deploys (disk-attached instance stops first). No backups. No monitoring/alerting/APM; console logging only; audit interceptor fail-open. Dockerfile: no `USER` (runs as root), no `HEALTHCHECK`, never runs migrations.

| ID | Issue | Effort | Impact | Priority |
|---|---|---|---|---|
| IN-01 | **No backups** of the disk that is the system of record + silent write-failure swallowing | Quick (nightly copy + throw-on-fail) | Total-loss scenario removed | **P0** |
| IN-02 | **No CI**; auto-deploy from main ungated; lint broken (script exists, no eslint config) | Quick–Small (template already at `docs/ci.workflow.txt` + `ai/templates/ai-docs-check.yml`) | Untested code stops shipping straight to prod | **P0** |
| IN-03 | Single point of failure; no zero-downtime deploys; single-instance lock-in (by JSON-store design) | Large | Accept short-term; solved by DB migration | P2 |
| IN-04 | No monitoring/error tracking; request logging absent (1 `console.log`) | Quick–Small | Incident diagnosis minutes not hours | P1 |
| IN-05 | Container runs as root; no image healthcheck | Quick | Hardening | P2 |
| IN-06 | Docs promise Postgres + migrations that don't exist (DEPLOY.md vs render.yaml) | Quick (docs) | Stops future misdeployment | P1 |
| IN-07 | 5 env vars undocumented in `.env.example` (incl. `STATE_DIR` — the one that decides where production data lives) | Quick | Deploy safety | P2 |

---

## 14. Security Assessment (dedicated review)

### 14.1 Findings by severity

| ID | Sev | Finding | Location | Fix | Effort | Priority |
|---|---|---|---|---|---|---|
| SEC-01 | **Critical** | Real staff usernames + weak pattern passwords hardcoded in source of a **public repo**; 30-day tokens; rotation overdue (project's own RISK-002/FACT-007). Prisma seed uses one shared password incl. SUPER_ADMIN | `staff-auth.module.ts:52-63`; `prisma/seed.ts:6` | Rotate live passwords NOW; remove from source; purge git history; force change on first login; consider making repo private | Quick–Small | **P0** |
| SEC-02 | **Critical (latent)** | `JWT_SECRET \|\| 'dev-secret'` fallback in 11 files — forgeable admin tokens if env unset | `jwt.strategy.ts:23`, `staff-auth:180`, `api-gate:69`, +8 | Fail-fast boot check; one shared JwtConfig module | Quick | **P0** |
| SEC-03 | High | `ApiGateGuard` **open when `PUBLIC_API_TOKEN` unset** — exposes paid Claude chat, Zoho financial dashboard, and GET+**PUT** `/api/appstate` (whole-dataset overwrite). Rate limiter keyed on spoofable `x-forwarded-for`, in-memory, resets on deploy | `api-gate.guard.ts:35-53` | Fail-closed in production; auth on appstate PUT; real client IP | Small | **P0** |
| SEC-04 | High | Public `POST /api/auth/register` accepts client-supplied `role`/`department`/`accessLevel` up to SUPER_ADMIN (dormant system, but mounted & routable) | `auth.service.ts:29`, `auth.dto.ts:34-42` | Strip privileged fields from public registration (or unmount System B) | Quick | **P0** |
| SEC-05 | High | Burned credentials: Zoho OAuth secret+refresh token and Anthropic key pasted into chats; rotation overdue | `CLAUDE.md:49-52`, handoffs | Rotate all; wipe shell history | Quick (owner) | **P0** |
| SEC-06 | High | Webhook/ingest auth is a plain static shared secret — no HMAC, no timestamp/replay protection; body may assert staff identity | `customer-portal:281`, `muhammed.controller:56-58` | HMAC-SHA256 + timestamp window; constant-time compare; role allow-list | Small | P1 |
| SEC-07 | Medium | Stored XSS: staff-app `esc()` misses `>`; 18 `innerHTML` sites render attacker-influenceable names (customer registration / ingest) | `apps/mobile-app/app.js:1119` | Complete `esc()`; audit interpolations; CSP | Small | P1 |
| SEC-08 | High (systemic) | `@Public()`-then-re-guard pattern: global guard bypassed on every live route; one forgotten decorator = open money endpoint (has happened before — bills) | all System-A controllers | Scoped global guard for System A; reserve `@Public()` for the true public trio | Medium | P1 |
| SEC-09 | High | Weak auth governs the money system: 4-char password minimum, 30-day non-refresh tokens, ad-hoc role checks | `staff-auth.module.ts:22`, customer-portal | Raise minimums; short TTL + refresh; centralize role checks | Medium | P1 |
| SEC-10 | Medium | JWTs + shared API key in `localStorage` (XSS-readable) across all apps | `app.js:748`, `order.js`, dashboards | Pairs with SEC-07/CSP; consider httpOnly session for admin surfaces | Medium | P2 |
| SEC-11 | Medium | No security headers (no helmet/CSP/HSTS); CORS allow-all when unset; Swagger `/docs` public (197 routes enumerated) | `main.ts` | helmet+CSP+HSTS; CORS fail-closed; gate /docs in prod | Quick | P1 |
| SEC-12 | Medium | Customer portal shares the staff data store — team's own docs say unsafe to expose publicly | DEPLOY-NTBFLLC.md:56-62 | Keep unlinked until per-customer isolation done | Medium | P2 |
| SEC-13 | Low | Static admin dashboard + committed Zoho org snapshot publicly served; infra IDs (org, Supabase ref) disclosed in `ai/*.md` of a public repo | `apps/admin-dashboard/` | Remove real snapshots; consider repo visibility | Quick | P2 |
| SEC-14 | Low | Audit log capped at 10 k rows (hash chain breaks on eviction), fail-open; exporter disabled | `audit/` | Rotate to dated files; enable exporter (post-backup) | Small | P2 |

### 14.2 Checklist coverage

- **Secrets management:** no live API keys committed (good); render `sync:false` pattern correct; the failures are hardcoded passwords (SEC-01) and burned creds (SEC-05).
- **AuthN/AuthZ:** §7 + SEC-02/03/04/06/08/09.
- **File upload security:** 15 MB base64 JSON bodies buffered in memory; photos on disk with role-checked serving (good); **no cleanup/retention** (DB-11); no content-type validation of decoded images.
- **API protections:** ValidationPipe whitelist+forbid (good); no helmet/CSP/HSTS; CORS fail-open; Swagger public (SEC-11).
- **Input validation:** strong on System-A DTOs; **bare `@IsObject()`** on bill/document payloads (OCR-02); no arithmetic checks.
- **Rate limiting:** one in-memory, spoofable, per-instance limiter (SEC-03). Nothing on login endpoints (brute-force viable against 4-char passwords).
- **Audit logging:** hash-chained interceptor is genuinely good; fail-open, 10 k cap, `anonymous` actors on ApiGate-only routes (ZH-07), export disabled (SEC-14).
- **Backup & disaster recovery:** **none** — no backup, no restore procedure, no drill (IN-01). RPO/RTO effectively infinite.
- **OWASP Top 10 observations:** A01 Broken Access Control — SEC-03/04/08, ZH-07 · A02 Crypto Failures — SEC-02 (static fallback secret), non-constant-time compares · A03 Injection — **no SQL/command injection or path traversal found** (parameterized ORM, no exec, no user paths); XSS SEC-07 · A04 Insecure Design — fail-open gates, no reconciliation loop · A05 Security Misconfiguration — SEC-11, root container, public Swagger · A06 Vulnerable Components — deps current; risk is what's absent (helmet, limiter, HMAC) · A07 Ident/Auth Failures — SEC-01/09, no login rate limits · A08 Software/Data Integrity — no CI, auto-deploy, unsigned webhooks · A09 Logging/Monitoring Failures — IN-04, SEC-14 · A10 SSRF — no user-controlled URL fetches found.

---

## 15. Performance & Scalability Review

| ID | Finding | Fix | Effort | Priority |
|---|---|---|---|---|
| PF-01 | Every mutation synchronously rewrites its **entire** JSON file on the request thread; bcryptjs (pure JS) on the event loop | Async I/O + per-store write queue; DB migration later | Small / Large | P1 |
| PF-02 | No pagination on any System-A list endpoint — payloads grow forever | `?limit=&before=` cursors | Small | P1 |
| PF-03 | Field app polls: full appstate blob every 6 s per device (+30 s badges, +20 s outbox) ≈ 1 req/s baseline | `rev`/ETag 304 short-circuit (server already tracks `rev`) | Quick | P1 |
| PF-04 | 15 MB base64 bodies buffered wholesale; photos served as base64 JSON | multipart upload; stream binary responses | Small | P2 |
| PF-05 | Linear scans on hot paths (`findByRef` per ingest; `allOrders` O(n·m); EOD walks everything) | In-store Map indexes | Quick–Small | P2 |
| PF-06 | Zoho 200-record truncation (also ZH-02) | Paged fetch | Quick | P1 |
| PF-07 | 272 KB uncompressed `catalog.js` on staff-app critical path, not precached; Leaflet from CDN breaks offline maps | Precache in service worker; vendor Leaflet | Quick | P2 |
| PF-08 | Single-instance lock-in by design (in-memory state, whole-file writes) | Documented constraint; resolved by DB migration | Large | P2 |

---

## 16. Code Quality Observations

- **TypeScript:** `strictNullChecks` on, but `noImplicitAny` **off**; money records typed `any[]` — a typo'd field compiles clean and silently corrupts financial data. (Medium, P2)
- **Lint:** `npm run lint` references eslint — **no eslint config or dependency exists**. (Quick, P1 — bundle with CI)
- **Mega-files:** `finance.module.ts` 866 lines; `app.js` 3,104 lines covering all 8 roles — the two highest-churn, highest-risk files. (Small/Medium, P2)
- **Duplication:** API-base expression ×7; `esc()` ×2 (the buggy copy is the staff one); JWT factory ×11; ingest-token check ×2; dashboard scaffold ×6; catalog data ×4.
- **Dead code:** ~2,600 LOC System B mounted and routable (DEC-018 approved its deletion — not executed); demo dashboards with fictional personas; orphaned agent module; one-time July backfill still mounted.
- **Logging:** essentially none (one `console.log`); no request logging or exception filter.
- **Tests:** 3 files / ~21 tests; zero on finance, orders, auth, guards. The "32/32 regression suite" is manual.
- **TODO inventory:** zero formal TODO/FIXME markers — deferred work lives in `ai/TASK_QUEUE.md` (24+ open tasks); in-code flags: "TEMPORARY passwords", attachment placeholder, "tighten before production" on dashboard, payroll "placeholder statutory deduction".

---

## 17. Doc-vs-Code Discrepancies (validate, don't assume)

| # | Claim | Reality |
|---|---|---|
| 1 | `README.md`/`IMPLEMENTATION_STATUS.md`: full ERP delivered ✅ | Those modules are dormant System B; **every one 500s in production** (no DB). The live system isn't mentioned in README at all |
| 2 | `TESTING.md`: "CI runs on every push (`.github/workflows/ci.yml`)" | **File does not exist**; no CI at all; template parked at `docs/ci.workflow.txt` |
| 3 | `TESTING.md`: `node --check apps/mobile-app/copilot.js` | `copilot.js` **does not exist** (deleted; its backend endpoint is orphaned — AI-01) |
| 4 | `DEPLOY.md`: blueprint provisions "a managed Postgres"; `prisma migrate deploy` step | `render.yaml` has **no database**; **no migration files exist** — the command no-ops |
| 5 | `ARCHITECTURE.md`: bills endpoints unauthenticated; finance lacks idempotency; only statusHistory audit | All three **stale** — since fixed (staff guards, clientRef, hash-chained audit). ARCHITECTURE.md pins an old commit |
| 6 | `CLAUDE.md`: Rashid module "in PR, NOT deployed" | Stale — wired in `app.module.ts:71` on main |
| 7 | Hard rule: Zoho org `928751913` on `.com` only | `render.yaml`, `.env.example`, code defaults still ship the **wrong org `170000198188` and `.ae` hosts** (ZH-04) |
| 8 | "Purchase bill photo → Zoho ✅" | Preview-only (writes locked); photo attachment is a fake-success stub |
| 9 | Zoho integration ✅ | Read-only in practice; reads truncate at 200; no sync; no entity mapping |
| 10 | WhatsApp bot part of platform (v16/v22/v41 in three docs) | Version contradictions; source outside repo, unauditable (WA-01) |
| 11 | Push notifications (TRD) | Log-only FCM stub |
| 12 | Model default docs | `claude-sonnet-5` (code) vs `claude-sonnet-4-6` (render/docs) |

---

## 18. Technical Debt Register (severity-ranked)

| # | Debt item | Severity | Effort | Business impact if unaddressed |
|---|---|---|---|---|
| D1 | No backups of `/var/data` | **Critical** | Small | Total, unrecoverable loss of live business records |
| D2 | Silent write-error swallowing in all stores | **Critical** | Quick | Invisible money-record loss under disk pressure (pairs with D1) |
| D3 | No CI; merge = production deploy; lint broken | High | Quick–Small | One bad merge takes down live trading |
| D4 | Hardcoded/burned credentials unrotated (public repo) | **Critical** | Quick | Full platform + accounting compromise |
| D5 | `'dev-secret'` fallback ×11; fail-open gates (`PUBLIC_API_TOKEN`, CORS) | High | Quick | Config slip = forgeable auth / open money API |
| D6 | No app→Zoho sync; live money un-booked | High | Large | Daily books drift; VAT exposure; manual catch-up compounds |
| D7 | System B dormant-but-routable (~80 dead endpoints, public Swagger) | Medium | Medium | Attack surface + misleads every engineer/doc reader |
| D8 | Catalog duplicated ×4, hand-synced | Medium | Small | Silent mispricing of real orders |
| D9 | Stored-XSS `esc()` gap in staff app | Medium | Quick | Admin session takeover via hostile customer name |
| D10 | Wrong Zoho org/DC in repo defaults | Medium | Quick | Env rebuild targets wrong ledger |
| D11 | Zero tests on money modules | High | Small–Medium | Every finance change re-risks cash handling; blocks D6 |
| D12 | Sync whole-file JSON persistence; no pagination; single-instance lock-in | Medium | Small→Large | Performance decay; no scaling path |
| D13 | Zoho 200-record cap | Medium | Quick | Wrong KPIs; bad bill matching |
| D14 | `attachToRecord` fake success | Medium | Small | Invisible audit-evidence gap |
| D15 | Mega-files, duplicated helpers, demo surfaces, stale root docs | Low-Med | Medium | Onboarding drag; recurring display bugs |
| D16 | Audit cap/eviction; in-memory dedupe & rate-limit state | Low | Quick–Small | Weakened tamper evidence |
| D17 | No order↔payment reconciliation; no invoicing in live pipeline | High | Medium | Revenue leakage invisible |
| D18 | No stock control in live pipeline; inventory in 4 disconnected places | High | Medium | Overselling; blind warehouse |
| D19 | No undo/correction mechanism | High | Medium | Permanent bad data from everyday mistakes |
| D20 | No scheduled-job infrastructure | Medium | Small | Blocks backups, sync, digests |

---

## 19. Risk Register

| ID | Risk | Likelihood | Impact | Mitigation (backlog ref) |
|---|---|---|---|---|
| R1 | Disk loss/corruption destroys all live business data | Medium | **Catastrophic** | P0-02 backups; P2 DB migration |
| R2 | Credential abuse (public repo passwords, burned tokens) | **High** | Critical | P0-01 rotation |
| R3 | Silent data loss via swallowed write errors on full disk | Medium | High | P0-02; DB-11 photo cleanup |
| R4 | Bad merge deploys straight to production | High | High | P0-06 CI |
| R5 | Cash misappropriation/typos via unbounded COD + order-less receipts + no reconciliation | Medium | High | P0-05, OL-03 |
| R6 | Duplicate/garbage ledger writes when Zoho writes are enabled | High (once enabled) | High | ZH-01, ZH-07, OCR-02 gate |
| R7 | Books drift: live revenue un-booked in Zoho | Certain (ongoing) | High | ZH-09 sync |
| R8 | WhatsApp bot unrecoverable / order silently lost in handoff | Medium | High | WA-01, WA-03 |
| R9 | AI cost blowout via open/orphaned endpoints | Medium | Medium | AI-01, AI-02, SEC-03 |
| R10 | Stored XSS → staff/admin session theft | Medium | Medium-High | SEC-07, SEC-11 |
| R11 | Prompt injection steers staff-facing AI answers | Low-Med | Medium | AI-04 |
| R12 | Config landmine deploys (wrong org, missing env) | Medium | Medium | ZH-04, SEC-02/03, IN-07 |
| R13 | UAE PDPL/privacy exposure (undisclosed AI processing) | Low-Med | Medium | OCR-05 |
| R14 | Bus factor 1 (single maintainer/agent; no runbook) | High | Medium | §20 docs |

---

## 20. Missing Documentation

1. **A truthful "start here"** — README describes the dead system; rewrite as a one-pager: what's live, System A vs B, where truth lives (`ai/`). (Quick)
2. **Operations runbook** — restore (once backups exist), deploy rollback, Zoho token rotation, disk-full response, bot rollback. Bus factor is 1. (Small)
3. **Environment variable reference** — required-vs-optional, failure behavior; fix the 5 undocumented vars and the wrong Zoho values. (Quick)
4. **System A API spec** — the frozen WhatsApp ingest contract has no standalone document; bot maintainers must read source. (Small)
5. **WhatsApp bot source + table schema vendored** (with WA-01). (Quick–Small)
6. **JSON store data dictionary** — field-level reference for the files that are the production database; prerequisite for sync & migration design. (Small)
7. **Stale-doc banners** — stamp README/ARCHITECTURE/IMPLEMENTATION_STATUS/DEPLOY/handoff as historical, pointing at `ai/`. (Quick)
8. **Testing guide refresh** — remove nonexistent CI/file references; document the real regression procedure. (Quick)

---

## 21. P0–P3 Prioritized Engineering Backlog

### P0 — Immediate (existential risk; ~1 focused week total)

| # | Item | Refs | Effort |
|---|---|---|---|
| P0-01 | **Rotate everything**: 4 staff passwords, admin password, Zoho OAuth set, Anthropic key; remove from source; purge git history; force password change on first login; decide repo visibility | SEC-01, SEC-05, ZH-05 | Quick–Small |
| P0-02 | **Nightly off-box backup of `/var/data`** (Supabase Storage per DEC-019) + restore drill; make store `save()` throw instead of swallowing; disk-space health check | IN-01, D1, D2 | Quick–Small |
| P0-03 | **Fail-closed configuration**: boot-fail on missing `JWT_SECRET`; `ApiGateGuard` refuses when `PUBLIC_API_TOKEN` unset in prod; CORS allowlist required; auth on `PUT /api/appstate` | SEC-02, SEC-03 | Quick |
| P0-04 | **Close privilege escalation**: strip role/department/accessLevel from public register; `StaffAuthGuard` + role on `/api/documents/*`; delete or gate the orphaned `/api/agent/chat` | SEC-04, ZH-07, AI-01 | Quick |
| P0-05 | **Cash-control quick fixes**: bound COD amount to order total (+tolerance, reason for short collection); require orderId/billAmount or force approval on order-less receipts | OL-01, OL-02 | Quick |
| P0-06 | **Activate CI** (build + tests + lint + `/ai` docs linter) gating deploys; fix eslint setup | IN-02, D3 | Quick–Small |
| P0-07 | **Fix wrong Zoho org/DC repo defaults**; extend org guard to reads | ZH-04, AI-05 | Quick |
| P0-08 | **Vendor the WhatsApp bot source** into the repo; document deploy config | WA-01 | Quick–Small |
| P0-09 | **Kill the fabricated-data fallback** (`mockExtract`) in the production bill flow | OCR-01 | Quick |

### P1 — Short term (30 days)

| # | Item | Refs | Effort |
|---|---|---|---|
| P1-01 | Zoho write-path readiness (gate to enabling writes): idempotency + duplicate checks, nested validation + arithmetic cross-checks, honest attachment (implement or report false), retry/backoff + 429 handling, pagination fix, dashboard cache | ZH-01/02/03/06, OCR-02 | Small×5 |
| P1-02 | WhatsApp reliability: bot-side delivery flag + retry loop; reconciliation view (wa_orders vs ingested); HMAC ingest auth; role allow-list on `/muhammed/wa`; persist Muhammed dedupe | WA-03, WA-06, AI-03, WA-02 | Small |
| P1-03 | Order-to-cash loop: auto-receivable on DELIVERED, receipt↔order reconciliation, unpaid-delivered report; admin override restraint | OL-03, OL-06 | Medium |
| P1-04 | Money-path test suite (FinanceService lifecycle, order transition matrix, guards) — prerequisite for sync work | D11 | Small–Medium |
| P1-05 | Security hardening batch: helmet/CSP/HSTS, Swagger gated, complete `esc()` + hostile-name test, login rate limiting, request logging + exception filter | SEC-07, SEC-11, IN-04 | Small |
| P1-06 | Scheduler infrastructure (`@nestjs/schedule`): first jobs = backup verify, unpaid-delivered digest | OL-11, D20 | Small |
| P1-07 | Lock + sanitize appstate (rev-check PUT, size/schema validation, keep N revisions); "tool data is untrusted" rule for Muhammed | AI-04, SEC-03 | Small |
| P1-08 | AI cost controls: real-IP rate keying, per-staff quotas, token-usage logging, budget alert | AI-02 | Small |
| P1-09 | Single-source catalog served from one endpoint + consistency check | OL-10, D8 | Small |
| P1-10 | Prisma migrations baseline committed + applied in entrypoint (or formally decide System-B deletion first, DEC-018) | DB-01, D7 | Small |
| P1-11 | Docs truth pass: README rewrite, stale banners, env reference, ingest-contract spec | §20 | Small |

### P2 — Medium term (90 days)

| # | Item | Refs | Effort |
|---|---|---|---|
| P2-01 | **App→Zoho daily sync** (the biggest gap): design with data dictionary + posted-ledger idempotency + `july-import` exclusion; then build | ZH-09, D6 | Large |
| P2-02 | Stock control in the live pipeline + single inventory source-of-truth decision | OL-04, OL-09, D18 | Medium |
| P2-03 | Undo/correction: time-boxed admin-approved compensating reversals for receipts/payments/COD | OL-05, D19 | Medium |
| P2-04 | Begin JSON→Postgres migration for money stores (schema exists); async I/O + pagination + Map indexes as interim | DB-10, PF-01/02/05 | Medium→Large |
| P2-05 | Execute System-B deletion (DEC-018) — removes ~80 dead endpoints, SEC-04 surface, doc confusion | D7 | Medium |
| P2-06 | Auth consolidation: one auth system, 8-char minimums, short TTL + refresh, centralized role guard, scoped global guard (kills the `@Public()` pattern) | SEC-08, SEC-09 | Medium |
| P2-07 | Notifications v1 (WhatsApp templates or FCM) for order status; customer-portal isolation review | OL gaps, SEC-12 | Small–Medium |
| P2-08 | Split mega-files; PDF bill support; multipart uploads; photo retention/cleanup; privacy-page disclosure update | D15, OCR-04/05, PF-04, DB-11 | Small×4 |
| P2-09 | Monitoring: error tracker + uptime alert + audit-log rotation & export enablement | IN-04, SEC-14 | Small |

### P3 — Long term (12 months)

| # | Item | Refs |
|---|---|---|
| P3-01 | Complete DB migration → multi-instance, zero-downtime deploys, real DR (tested RPO/RTO) | PF-08, IN-03 |
| P3-02 | Procurement→inventory→Zoho item lifecycle (rebuilt on the live system, not System B); barcodes, item-level analytics (TASK-004/013) | OL-08/09 |
| P3-03 | AI orchestrator (only if DEC-014's five gates pass); AI cost dashboards | ai/ROADMAP |
| P3-04 | Compliance program: UAE PDPL data map, retention policies, subprocessor disclosures, security review cadence | OCR-05, R13 |
| P3-05 | TypeScript strict mode + typed store records across the money paths | §16 |

---

## 22. Actionable Roadmap

**Next 7 days (P0):** rotate credentials → stand up nightly backups + fail-loud writes → fail-closed config (JWT/API gate/CORS) → close the four open-door endpoints → bound COD + receipts → turn on CI → fix Zoho org defaults → vendor bot source → kill mock-data fallback. *Everything here is Quick/Small; combined this removes the majority of existential risk.*

**30 days (P1):** Zoho write-path readiness (do NOT flip `ZOHO_WRITES_ENABLED` before P1-01 lands) · WhatsApp order-handoff reliability · order-to-cash reconciliation · money-path tests · security hardening batch · scheduler + appstate lock · AI cost controls · single-source catalog · docs truth pass.

**90 days (P2):** build the app→Zoho daily sync · live stock control · correction/reversal mechanism · begin JSON→Postgres migration · delete System B · auth consolidation · notifications v1 · monitoring.

**12 months (P3):** complete the migration (multi-instance, zero-downtime, tested DR) · rebuild procurement/inventory on the live system · orchestrator behind its gates · PDPL compliance program · strict typing across money paths.

---

## 23. Positive Findings (preserve these)

- Zoho write choke point: fail-closed flag + hard org allow-list — prevented a wrong-org incident already.
- Server-side re-pricing: order paths never trust client or LLM prices.
- Frozen, idempotent WhatsApp ingest contract with needsReview gating.
- Dual-control money flows: admin-approved payments, recipient-confirmed transfers, discount approval queue.
- Hash-chained audit interceptor with verify endpoint.
- `origin:'july-import'` live-vs-historical separation, enforced consistently.
- The `ai/` governance protocol — evidence levels, append-only decisions, honest risk register. Better than most professional teams. Most of this audit's findings corroborate risks the project already recorded; the failure mode is enforcement, not awareness.

## 24. Methodology & Limitations

Seven parallel read-only agents audited the codebase with mandatory `file:line` evidence; findings were deduplicated and re-prioritized centrally on one P0–P3 scale. Limitations: (1) the WhatsApp bot source is outside the repo — webhook verification, templates, media, and session-window handling are **unverifiable** until WA-01 lands; (2) runtime environment values (whether `PUBLIC_API_TOKEN`/`CORS_ORIGIN` are actually set in Render) could not be inspected — fail-closed fixes are recommended regardless; (3) dependency review was static (no `npm audit` run against live lockfiles); (4) shallow clone — full git history (e.g., for secret-leak archaeology) was not examined.
