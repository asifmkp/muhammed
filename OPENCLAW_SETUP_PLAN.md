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
