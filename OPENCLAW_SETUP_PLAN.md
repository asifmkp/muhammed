# OpenClaw Personal Assistant — Setup & Usage Plan (v2, corrected)

Owner: Asif — runs **NTBF (National Trading)**, a foodstuff/FMCG wholesale business
in Ajman, UAE. Currency AED, timezone Asia/Dubai.
Scope of this plan: **Asif's PERSONAL assistant track only** — a productivity agent
on his own personal WhatsApp number via OpenClaw. It does NOT serve or extend the
NTBF business platform (repo `asifmkp/ntbf-platform`), which has its own live
WhatsApp bot, ops assistant scaffold, and document pipeline.

Current state: OpenClaw gateway runs on Asif's Windows Lenovo laptop (also host of
the existing desktop agent "Glitch" / SYS-07 — Glitch IS agent `main`). Android
node app paired (1/1 nodes). Model auth confirmed: Claude Max subscription login
(claude-cli runtime, primary anthropic/claude-opus-4-7), no API key. WhatsApp
plugin @openclaw/whatsapp@2026.7.1 installed; QR link + pairing pending.
The "Companion" desktop app is a single-instance tray app (OpenClaw.Tray.WinUI) —
its UI lives behind the system-tray icon; double-clicking the launcher is a no-op.

---

## Standing Guardrails (adopted from the NTBF platform session — read first)

1. **Zoho org guard:** the ONLY authorized Zoho Books organization is org ID
   `928751913` (US .com DC). The second org (ID starting `170000198188`, .ae DC)
   is FORBIDDEN — no reads, no writes, ever, absent Asif's own explicit words.
2. **No Zoho writes from this stack, ever,** without Asif's explicit per-action
   instruction. The platform already has a gated document→Zoho pipeline; a second
   independent Zoho-writing path is the exact class of mistake that created the
   forbidden org.
3. **Business WhatsApp number is off-limits.** The business runs a production
   WhatsApp bot (Supabase edge function + 360dialog) on the business number for
   customers and staff. This assistant links ONLY to Asif's personal number.
4. **The 3 existing cron jobs are Glitch's** (confirmed 2026-07-27, all owned by
   `agent:main`): `ntbf-health-watchdog` (5-min read-only NTBF health poll),
   `ashi-checkin-30m` (task sweep, Dubai waking hours), `cockpit-daily-sweep`
   (08:00 Dubai). Rule: **never create or modify jobs under `agent:main`** —
   personal-assistant automations get their own agent identity.
5. **Briefing overlap:** the platform has "Muhammed ops Phase A" (owner-gated
   daily-briefing generator, currently inert). Any business-data briefing from
   this assistant duplicates it — that choice belongs to Asif explicitly.
6. **Agent↔agent communication goes through Asif** — nothing direct. Chat memory
   is never treated as verified; facts come from repos or live systems.
7. Secrets by env-var NAME only, never values. Nothing destructive,
   outward-facing, or money-touching without owner confirmation.
8. **Live vs Historical (platform sacred rule, respected here too):** July-2026
   records with origin `july-import` are reference-only — never treat them as
   live data in any lookup this assistant performs.

**Safe scope for this assistant** (per cross-session agreement): personal
briefings from Calendar/Gmail/Drive, reminders, personal document filing to
Drive, and read-only lookups. Anything touching business systems is a
cross-project decision Asif must state explicitly, recorded on both sides.

---

## Phase 0 — Foundations & Security (Day 1, ~30 min)

- [ ] Gateway host is the Windows Lenovo laptop — confirm it stays on (disable
      sleep-on-lid-close) or plan the Phase 6 always-on move.
- [ ] **Coexistence check with Glitch:** confirm with Asif (and Glitch) that this
      personal-assistant work shares the gateway cleanly — same gateway, separate
      concerns; do not modify Glitch's config, docs lanes, or cron jobs.
- [ ] Never expose the gateway port to the internet; use Tailscale/VPN if remote
      web-UI access is needed.
- [ ] DM policy: pairing/allowlist, personal number only (config in Phase 1).
- [ ] Approval rules: read/summarize/monitor/draft = automatic;
      send/create/pay/delete = always ask. (Matches platform standing rules.)

**Success check:** guardrails above acknowledged; laptop stays on overnight.

---

## Phase 1 — WhatsApp Channel, personal number ONLY (Day 1, ~15 min)

On the gateway (Lenovo laptop):

```bash
openclaw plugins install clawhub:@openclaw/whatsapp
openclaw channels login --channel whatsapp
```

QR appears in terminal → on the phone holding the assistant's WhatsApp account:
**WhatsApp → Settings → Linked Devices → Link a Device** → scan.

**DECIDED (2026-07-27):** the assistant uses a **dedicated work number** — not
the 360dialog business number (production bot lives there) and not Asif's
personal number. The bot appears as a normal contact Asif messages from his
own phone. Number itself is kept out of this repo.

Lock down `openclaw.json` (E.164 format):

```json5
{
  channels: {
    whatsapp: {
      dmPolicy: "pairing",
      allowFrom: ["+971XXXXXXXXX"],   // Asif's personal number
      groupPolicy: "disabled"
    }
  }
}
```

Approve first pairing if prompted: `openclaw pairing list whatsapp` →
`openclaw pairing approve whatsapp <CODE>`.

**Success check:** "hello" in self-chat gets a reply from agent `main`.

---

## Phase 2 — Context / Memory (Day 1–2, ~20 min)

Standing context to give the agent (chat: "Remember this permanently"):

> I'm Asif. I run NTBF (National Trading), a foodstuff/FMCG wholesale business
> in Ajman, UAE. Currency AED, timezone Asia/Dubai. This assistant is my
> PERSONAL productivity agent: Calendar, Gmail, Drive, reminders, personal
> documents. It does not run business operations — the NTBF platform does that.
> Zoho Books: read-only lookups at most, only org 928751913, and NEVER any
> write without my explicit instruction in that moment. Never message anyone
> but me without approval. Short replies, max 10 lines unless asked.

Test read-only: today's calendar, unread email summary, a Drive file search.

**Success check:** all answer correctly without re-explaining who Asif is.

---

## Phase 3 — Personal Automations (Week 1)

Start with ONE (personal scope only):

1. **Morning briefing, 7:00 AM Asia/Dubai:** unread-important emails summarized
   + today's calendar + open reminders. Under 12 lines.
   - *Zoho/business content in the briefing is EXCLUDED for now* — the platform's
     Muhammed ops briefing (currently inert) covers that ground. If Asif wants
     business figures in a daily message, he chooses which system delivers it,
     and the decision gets recorded on both sides.

2. *(Optional, later)* Personal reminders/watchers (renewals, personal payments,
   a website/price watch — nothing business-operational).

**Before creating any cron job:** the 3 existing jobs are Glitch's (`agent:main`)
— identified, do not touch. Personal-assistant jobs must run under a separate
agent identity (name to be agreed with Asif/Glitch), never under `agent:main`.

**Success check:** briefing arrives 3 days in a row; Glitch's jobs untouched.

---

## Phase 3B — PERSONAL Document Filing (WhatsApp → Drive only)

Reduced scope after cross-session correction: **business records (supplier
bills, POs, delivery notes, cash/purchase vouchers) belong to the platform's
existing capture pipeline** — do not build a parallel one, and this assistant
never posts documents to Zoho.

What remains in scope — personal/administrative documents:

```
WhatsApp attachment (personal doc: passport/ID, insurance, licenses,
personal receipts, vehicle/house papers)
   → agent reads/OCRs
   → renames: YYYY-MM-DD_<category>_<entity>_<doctype>.<ext>
   → files to Drive: /Personal/<HR-ID | Insurance | Government | Vehicles | Misc>/
   → logs one row in /Personal/Document-Index
   → replies with filename + link
```

- If a document looks like a **business record**, the agent files nothing and
  replies: "This looks like a business document — send it through the staff
  platform capture instead." (One clarifying question if unsure.)
- PII (passports/IDs): restricted Drive folder, never re-shared or forwarded
  without explicit approval; Google account must have 2FA.
- Never overwrite/delete existing files without asking.

**Success check:** a personal insurance PDF gets filed and indexed; a supplier
bill photo gets correctly REDIRECTED to the platform, not filed.

---

## Phase 4 — Approval Gates (Week 1, in parallel)

- [ ] Send/create/pay/delete actions require explicit confirmation (app or chat).
- [ ] "Draft first, show me" habit for any outbound message.
- [ ] Test: ask it to email Asif himself — must pause for approval (Approvals
      card in Android app shows pending item).

**Success check:** nothing outbound happens without a visible approval step.

---

## Phase 5 — Daily Workflow (ongoing)

| Situation | Tool |
|---|---|
| Quick personal task/question on the go | WhatsApp self-chat (voice notes) |
| Approvals, status, files | OpenClaw Android app |
| Business operations (orders, cash, Zoho, staff) | NTBF platform / staff PWA — NOT this assistant |
| Architecture/design docs | Glitch (Lane 1, desktop) |
| Building/configuring this assistant | This Claude Code session |
| Thinking/review | Claude chat |

---

## Phase 6 — Later Upgrades (only when a real limit is hit)

- Always-on host for the gateway (mini-PC/VPS) if the laptop being off causes
  missed briefings — coordinate with Glitch's hosting needs, same machine today.
- Telegram as backup channel.
- Any business-system access for this assistant = explicit cross-project
  decision by Asif, recorded in both this repo and the platform's /ai knowledge
  base.

## What NOT to do

- No writes to Zoho Books — ever — without Asif's explicit per-action words.
- No reads/writes on the forbidden Zoho org (`170000198188...`), period.
- Never link or send through the business WhatsApp number.
- Don't duplicate platform systems (briefing, document capture) — flag overlaps
  to Asif instead.
- Don't modify or delete the 3 pre-existing cron jobs.
- No secrets, phone numbers, or credentials in this repo.
