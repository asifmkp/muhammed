# OpenClaw Personal Assistant — Detailed Setup & Usage Plan

Owner: Asif
Current state: Gateway installed and **Healthy**, Android node app connected (1/1 nodes online), agent `main` running with 3 cron jobs. WhatsApp channel not yet linked.
Goal: A secure, always-on AI assistant that manages WhatsApp, email, calendar, and Zoho Books — controllable from phone and laptop.

---

## Phase 0 — Foundations & Security (Day 1, ~30 min)

Do this BEFORE connecting WhatsApp. An always-on agent with loose security is a liability.

- [ ] **Confirm where the gateway runs** (laptop / desktop / server). That machine must stay powered on for the assistant to work while you're out.
  - If it's a laptop: disable sleep-on-lid-close, or plan the Phase 6 VPS move.
- [ ] **Never expose the gateway port to the internet.** For remote access to the web UI, use Tailscale (free) or a VPN.
- [ ] **Set the DM policy to pairing/allowlist** in `openclaw.json` (exact config in Phase 1) so only your number can command the agent.
- [ ] **Decide the approval rules** (enforced in Phase 4):
  - Auto-allowed: read, search, summarize, monitor, draft.
  - Always ask first: send email/message, create/send Zoho invoices, any payment, any delete.

**Success check:** you can state where the gateway runs and confirm it stays on overnight.

---

## Phase 1 — WhatsApp Channel (Day 1, ~15 min)

Run these on the **gateway machine** (not the phone app):

```bash
# 1. Install the WhatsApp plugin
openclaw plugins install clawhub:@openclaw/whatsapp

# 2. Log in — a QR code appears in the terminal
openclaw channels login --channel whatsapp
```

On the phone: **WhatsApp → Settings → Linked Devices → Link a Device** → scan the QR.
Note: the QR expires fast. If the gateway is headless, view its terminal live (SSH from phone) or use the app's **Settings → Channels** screen if your build shows the QR there.

**Number choice:**
- Personal number (start here): auto-enables self-chat mode — you talk to the agent in your "Message yourself" chat. Zero extra cost.
- Dedicated second number (upgrade later, Phase 6): cleaner separation, bot appears as a separate contact.

**Lock it down** in the gateway's `openclaw.json` (E.164 format):

```json5
{
  channels: {
    whatsapp: {
      dmPolicy: "pairing",
      allowFrom: ["+91XXXXXXXXXX"],
      groupPolicy: "allowlist",
      groupAllowFrom: ["+91XXXXXXXXXX"]
    }
  }
}
```

Approve your own first message if pairing mode asks:

```bash
openclaw pairing list whatsapp
openclaw pairing approve whatsapp <CODE>
```

**Success check:** you send "hello" on WhatsApp and agent `main` replies.

---

## Phase 2 — Teach It Who You Are (Day 1–2, ~20 min)

Short phone messages only work if the agent already has context.

- [ ] Add a context/memory note (via the agent's workspace context file, or just tell it in chat: "Remember this permanently: ..."):

> I'm Asif. I run a book business in Kerala (National Books). I use Zoho Books for
> accounting, Gmail for email, Google Calendar for scheduling. Currency is INR,
> timezone IST. Prefer short replies on WhatsApp — max 10 lines unless I ask for
> detail. Always show me drafts before sending anything. Never send an invoice or
> email without my explicit "send".

- [ ] Verify connected integrations from the app or chat: Gmail, Google Calendar, Google Drive, Zoho Books.
- [ ] Test each with a read-only question:
  - "Summarize my unread emails from today."
  - "What's on my calendar this week?"
  - "How many unpaid invoices this month in Zoho, and the total?"

**Success check:** all three answer correctly without asking who you are.

---

## Phase 3 — Core Automations (Week 1)

Set these up by simply messaging the agent. Start with two; add more only after a week of stable running.

1. **Morning briefing (highest value — do first):**
   > "Every day at 7:00 AM IST, send me on WhatsApp: (1) important unread emails
   > summarized, (2) today's calendar, (3) Zoho: any invoices due or overdue today.
   > Under 12 lines."

2. **One monitor you actually care about**, e.g.:
   > "Every evening at 6 PM, check Zoho for invoices that became overdue today and
   > list them. If none, stay silent."

3. *(Later)* Weekly business summary, Friday 5 PM: sales week-over-week, top overdue customers, cash position.
4. *(Later)* Price/stock/website watchers: "Check X every 6 hours, message me only if condition Y."

**Rules for good automations:** silent when nothing to report; every alert must be actionable; review after 2 weeks and delete any you ignore.

**Success check:** the 7 AM briefing arrives 3 days in a row and you actually read it.

---

## Phase 3B — Document Intake & Filing (WhatsApp → Cloud / Zoho)

Goal: send ANY document on WhatsApp (photo or PDF) and have the agent read it,
classify it, rename it, file it in the right place, and confirm back — no manual
sorting ever.

### The pipeline

```
WhatsApp attachment
   → agent reads / OCRs it
   → classifies: HR | Finance | Sales | Admin | Government | Personal-ID
   → renames: YYYY-MM-DD_<category>_<entity>_<doctype>.<ext>
        e.g. 2026-07-26_Finance_NationalBooks_PurchaseInvoice.pdf
             2026-07-26_HR_Rahul-K_Passport.pdf
   → uploads to the right folder / attaches to the right Zoho record
   → replies with: category, filename, storage link, and any extracted key data
   → logs one row in a running index sheet (date, type, entity, amount, link)
```

### Storage map (decide once, then it's automatic)

| Document type | Where it goes |
|---|---|
| Purchase/sales invoices, bills, receipts | Google Drive `/Business/Finance/<year>/<month>/` + attach to the matching Zoho Books record (bill/invoice/expense) when one exists |
| Asset purchases | Drive `/Business/Finance/Assets/` + Zoho Books fixed-asset record |
| Employee docs (passports, IDs, contracts, insurance) | Zoho People (HR app) against the employee's record if connected; otherwise Drive `/Business/HR/<employee-name>/` (restricted folder) |
| Government docs (GST, licenses, filings) | Drive `/Business/Government/<year>/` |
| Sales docs (quotes, POs from customers) | Drive `/Business/Sales/<customer>/` |
| Admin (rent, utilities, misc contracts) | Drive `/Business/Admin/<year>/` |
| Unclear / low confidence | Drive `/Business/_Unsorted/` — agent asks you one clarifying question |

### Setup steps

- [ ] Create the Drive folder skeleton above (or ask the agent to create it).
- [ ] Confirm the gateway has Google Drive access; test: "Create a test file in /Business/_Unsorted and send me the link."
- [ ] Zoho Books attachment test: "Attach this PDF to bill/invoice X."
- [ ] Zoho People (HR) is a **separate connector** from Zoho Books — connect it if
      you want employee docs filed there; until then the HR Drive folder is the fallback.
- [ ] Create the index sheet: `/Business/Document-Index` (date, category, entity,
      doctype, amount if any, link).
- [ ] Save the standing instruction (paste once into the agent chat):

> Standing rule: whenever I send a file or photo of a document on WhatsApp,
> read it, classify it (HR / Finance / Sales / Admin / Government / Personal-ID),
> rename it as YYYY-MM-DD_category_entity_doctype, file it per my storage map,
> attach invoices/bills to the matching Zoho Books record when one exists,
> add a row to the Document-Index sheet, and reply with what you did and the link.
> If you're less than ~80% sure of the category, put it in _Unsorted and ask me.
> Never overwrite or delete an existing file without asking.

### Sensitive-document rules (passports, IDs, insurance)

- These are PII. Keep them in a **restricted** Drive folder (or Zoho People) —
  never in generally-shared folders, and the agent must **never re-share or
  forward them** without explicit approval.
- WhatsApp media is end-to-end encrypted in transit, but the file then lives on
  your gateway machine and cloud storage — make sure both have disk encryption
  and your Google account has 2FA.
- Auto-filing is allowed; **deleting, sharing, or moving out of restricted
  folders always requires approval** (extends the Phase 4 gates).

**Success check:** send a photo of any invoice on WhatsApp → within a minute you
get back "Filed: Finance → 2026-07-26_..._PurchaseInvoice.pdf, attached to Zoho
bill #123, link: ..." and the index sheet has a new row.

---

## Phase 4 — Approval Gates (Week 1, in parallel)

Turn the Phase 0 decisions into enforced behavior:

- [ ] Configure OpenClaw so send/create/pay/delete actions require confirmation (approval prompt in the app or WhatsApp).
- [ ] Build the "draft first" habit: phrase requests as *"Draft a reply to X... show me before sending."*
- [ ] Test the gate: ask it to send a trivial email to yourself — confirm it pauses for approval and the Approvals card in the app shows the pending item.

**Success check:** the agent cannot send anything without a visible approval step.

---

## Phase 5 — Daily Workflow (ongoing)

| Situation | Channel | Example |
|---|---|---|
| Quick question / task on the go | WhatsApp (voice note!) | "Any important emails since noon?" |
| Approvals, status, files | Android app | Approve a draft; browse workspace files |
| Building new automations, fixing configs | Claude Code (laptop / this session) | "Add a supplier-payment reminder cron" |
| Thinking, planning, second opinions | Claude chat | "Review this pricing email for tone" |

Habits that compound:
- Voice-note tasks instead of typing — fastest input on a phone.
- Forward emails/photos/receipts into the chat: "extract the amount and vendor, log it."
- End of day: "What did you do today? Anything pending my approval?"

---

## Phase 6 — Later Upgrades (Month 2+, only when a real limit is hit)

- **Always-on box:** move the gateway from laptop to a cheap VPS or a mini-PC/Raspberry-class device at home. Symptoms it's time: missed briefings because the laptop was off.
- **Dedicated WhatsApp number:** second SIM/eSIM once the assistant is part of daily business.
- **More channels:** Telegram as a backup channel (3-minute setup) in case WhatsApp relinks are needed.
- **Second model (Gemini/Codex):** skip unless you start doing large coding projects and want an independent cross-model reviewer. For business admin, one brain + human approval gates is the right architecture.

---

## What NOT to do

- Don't give the agent open-ended send/pay permissions "to save time" — the 5 seconds per approval is the product, not friction.
- Don't create 10 cron jobs in week 1 — two good ones beat ten noisy ones.
- Don't expose the gateway to the public internet, ever.
- Don't put API keys or the `openclaw.json` allowlist file in any public repo.

## Timeline at a glance

| When | Milestone |
|---|---|
| Day 1 | Security decisions + WhatsApp linked + context note saved |
| Day 2–3 | Integrations tested read-only; morning briefing live |
| Week 1 | Approval gates verified; one monitor running |
| Week 2 | Review: delete ignored alerts, tune the briefing |
| Month 2+ | VPS / dedicated number / extra channels as needed |
