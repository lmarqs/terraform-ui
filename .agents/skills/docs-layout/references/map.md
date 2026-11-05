# The map

`docs/README.md` is the page a reader or an agent opens first: the kinds of page this repository keeps, when a fact
belongs to each, and what a good page of each achieves.

## Where a fact goes

The kind is the question a reader brings to a fact, never the fact's subject: one subject feeds several kinds. In
the made-up `acme`, the session timeout is a feature rule, the shared session store is a decision, and clearing
sessions is a runbook. Each fact has one owner; other pages link it, since a copy drifts, with link text that names
the fact, as `wait out [the rate limit](api/orders.md)`.

## What a page holds

- What the repository shows nowhere else: a reason, a rule, a convention and its scope. Code, contracts and history
  hold their own facts, and a page links them.
- What is true now. A tracker holds plans and state that change faster than a page.
- Reasons told as **what happened**: the situation, what it cost, what was tried, then the conclusion. "We need a
  queue" is a conclusion; "the payment call timed out and customers who retried were charged twice" is a reason a
  reader can test.

## Front matter

Front matter describes the page, so a reader picks it from a listing unopened; the body opens with content.
`description`: one sentence, at most 160 characters, opening with a verb naming the kind, as "Records why", unique.
`audience`: who opens it and when, at most 100 characters. `keywords`, optional: search words the body lacks, as an
error text. Write it last.

## Terms and guides

`docs/glossary.md` gives each concept one name: `**Term**:`, a definition of a sentence or two, then `_Avoid_:` with
the synonyms rejected. A kind of many pages has a guide, `docs/<kind>/README.md`, listing each page with a sentence.
Both appear with their first content: empty structure says something exists that doesn't.

## Misfilings

Most misplaced facts are one of these:

- A feature rule written as a decision. Giveaway: its options had to be invented.
- A decision filed under the feature it touches. Giveaway: a product or technology name with the reason it beat
  another.
- A screen or output convention written into a feature page. Giveaway: it names a component, a color or a position.
- A feature's behavior written as a standard. Giveaway: it applies to one unit only.
- A procedure written as a feature. Giveaway: numbered steps a responder follows.
- A copy of the tracker's state. Giveaway: a phase or status updated by hand. A team that keeps a plan in `docs/`
  for its reasons declares a kind for it and keeps the state in the tracker.

## A new kind

Which part of each kind's description do the facts miss? Do they have another owner, as an FAQ's answers
belong to the features they explain? Are they true now, and shown nowhere else? Can a good page of the kind be
described, so a review can compare a page with it?

## A good map

It says, for each kind the team keeps, when a fact belongs there and what a good page achieves, and links the
kind's pages once they exist. It names no section a page must have. A worked example, one subject of this repository
whose facts land in different kinds, shows why each lands where it does. It names the misfilings this team meets.
Its language, headings and order are the team's.
