# Status Check Prompt for ChatGPT Chats

Copy the prompt below and paste it into **each** ChatGPT conversation (or Project)
you want a report from. ChatGPT will answer based on that chat's own history.

---

## The prompt (copy everything between the lines)

```
You are now acting as my project auditor. Carefully re-read the ENTIRE
history of this conversation from the very first message to the last,
then produce a complete status report using EXACTLY the structure below.
Do not skip any section. If a section doesn't apply, write "None".

═══════════════════════════════════════
1. CHAT IDENTITY
═══════════════════════════════════════
- Suggested name for this chat (5 words max):
- Topic/category (e.g., business, accounting, coding, personal, learning):
- Approximate date range of our work here (first to last message):

═══════════════════════════════════════
2. PURPOSE & GOAL
═══════════════════════════════════════
- Why was this chat started? What problem was I trying to solve?
- What is the final outcome/deliverable I wanted?
- Has the goal changed or expanded since the chat began? If yes, how?

═══════════════════════════════════════
3. CURRENT STATUS
═══════════════════════════════════════
- Overall status (pick ONE): Not Started / In Progress / Blocked /
  Completed / Abandoned
- Progress estimate: ____% complete
- What exactly was the LAST thing we were working on when the chat
  stopped?

═══════════════════════════════════════
4. WORK COMPLETED
═══════════════════════════════════════
List everything that was finished in this chat, as bullets. For each:
- What was done
- Any output produced (document, plan, code, calculation, draft, etc.)

═══════════════════════════════════════
5. DECISIONS MADE
═══════════════════════════════════════
List all important decisions, choices, or conclusions we reached, so I
don't have to re-decide them later.

═══════════════════════════════════════
6. PENDING TASKS (most important section)
═══════════════════════════════════════
List ALL unfinished work as a numbered list. Include:
- Tasks I said I would do but never confirmed doing
- Tasks you suggested that we never started
- Half-finished items
For EACH pending task give:
  a) Task description
  b) Priority: High / Medium / Low
  c) Estimated effort: Quick (under 30 min) / Medium / Large
  d) What is needed to complete it

═══════════════════════════════════════
7. BLOCKERS & WAITING ON
═══════════════════════════════════════
- Is anything blocked? By what?
- What information, files, decisions, or actions are needed FROM ME
  before work can continue?
- Is anything waiting on a third party (bank, supplier, customer,
  developer, etc.)?

═══════════════════════════════════════
8. RISKS & DEADLINES
═══════════════════════════════════════
- Any deadlines mentioned in this chat (with dates)?
- Anything time-sensitive or at risk if ignored (penalties, expiry,
  lost opportunity)?

═══════════════════════════════════════
9. OPEN QUESTIONS
═══════════════════════════════════════
Questions that were raised in this chat but never answered.

═══════════════════════════════════════
10. RECOMMENDATION
═══════════════════════════════════════
- Single most important NEXT ACTION (one sentence, be specific):
- Should this chat be: Kept active / Merged with another topic /
  Archived as done / Deleted? Why?
- If I only had 15 minutes for this topic today, what should I do?

═══════════════════════════════════════
11. ONE-LINE SUMMARY
═══════════════════════════════════════
Finish with one line in this exact format:
[Chat name] | [Status] | [% complete] | [Top pending task] | [Priority]

Be honest and specific. Use only information actually present in this
conversation — do not invent details.
```

The one-line summary in section 11 is designed to be collected from every
chat into a single list — paste all the collected lines back to Claude for
one combined overview of the whole section.

---

## How to use it

1. Open ChatGPT and go to your section/folder of chats.
2. Open a chat, paste the prompt, send it.
3. Copy ChatGPT's answer into `STATUS-TRACKER.md` (one row/section per chat).
4. Repeat for each chat.
5. If you paste all the collected reports back to me (Claude), I can
   summarize everything into one overview: what's done, what's pending,
   and what to focus on next.

**Tip for ChatGPT Projects:** if your chats are grouped in a Project, you can
paste the prompt once in a new chat inside that Project and add: "Consider all
files and chats in this project," though per-chat results are more reliable.
