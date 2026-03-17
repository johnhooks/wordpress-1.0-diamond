# Trusted Sites

**Status: Pipedream. The concept is interesting, the mechanics are
entirely open. This is an evolution of the blogroll idea — there's
something here worth exploring, but nothing here is close to buildable.**

## The Problem

The blogroll was WordPress's way of saying "these are blogs I like." RSS
was "I can read your public posts." Neither created any real connection
between sites. They were one-directional and passive.

Meanwhile, the open web's default — public means anyone can read — has
become "public means anyone and any machine can consume your words for
any purpose." Scrapers, AI training sets, content farms. "Public" used
to mean sharing with people. Now it means surrendering to systems.

Federation (Mastodon, Bluesky) tries to fix distribution but asks site
owners to run complex infrastructure. Most people can't self-host a
Mastodon instance. The bar is too high.

Press already has a permission system that controls who can view, comment,
and edit content. The question is: can that system extend across site
boundaries?

## The Idea

Two Press site owners choose to trust each other. This is a bilateral
agreement — both sides opt in. Once established, users from one site can
view and comment on the other site's content, according to permissions
the site owner sets.

This is not federation. Federation tries to replicate the entire social
experience across instances. This is narrower: **I trust your site. By
extension, I trust the people you've curated. They can engage on my
site as if they had an account here, at the permission level I choose.**

### What it is

- A bilateral trust link between two independent Press sites
- Users from a trusted site can view shared content and leave comments
- The site owner controls the permission level granted to trusted users
- Each site remains fully sovereign — own database, own users, own content
- Trust is explicit and revocable

### What it is not

- Not federation — no shared timelines, no activity streams, no inbox
- Not single sign-on — users don't get accounts on the remote site
- Not content replication — posts stay on the origin server
- Not admin access — trusted users can never edit, manage, or administer
- Not automatic — both site owners must agree to the link

## How It Might Work

### Establishing Trust

Alice runs a Press blog. Bob runs a Press blog. They want to link their
sites.

1. Alice generates a trust invitation from her admin panel
2. She sends Bob the invitation (URL, email, however)
3. Bob accepts the invitation from his admin panel
4. Both sites exchange public keys
5. A bilateral trust link is established

Each site stores the other's public key and the agreed permission level.
No central authority, no third-party service.

### The Trust Tuple

In the existing permission model, a trusted site creates a group on the
destination. Alice's site has a group called "Bob's Blog" (or whatever
she names it). That group gets the permission level Alice chooses:

```
group:bobs-trusted → commenter → type:post
```

Users who arrive through the trust link and create accounts are added
to that group via the normal tuple:

```
user:charlie → member → group:bobs-trusted
```

Alice manages one group and its permissions. The trust link controls
who gets into that group. Standard tuple mechanics from there.

### Vouched Sessions and Account Creation

No identities are shared between sites. The origin site vouches for
the user — "this person is one of mine" — and the destination site
decides what level of access that earns.

**Viewing (no account required):**

1. Charlie on Bob's site clicks a link to Alice's shared post
2. Alice's site sees the request comes from a trusted site
3. Alice's site redirects to Bob's site with a callback: "vouch for
   this person"
4. Bob's site authenticates Charlie (they're already logged in) and
   confirms: "yes, charlie is one of mine"
5. Alice's site sets a vouched session cookie
6. Charlie can view shared content — no account, no signup, no friction

The vouch gets you in the door. A session is enough to read. This is
the lowest-friction path — the reader clicked a link and can see the
content. They didn't sign up for anything.

**Engaging (account required):**

When Charlie wants to do more than view — comment, interact, whatever
Alice's site allows for Bob's trusted group — Alice's site prompts
them to create a local account. The vouched session carries over:

1. Charlie clicks "Post Comment"
2. Alice's site says: "create an account to comment"
3. Charlie picks a display name, done
4. The new account is added to the trusted group for Bob's site
5. Charlie can now engage at whatever permission level Alice configured

The site owner decides what requires an account. Viewing might be free
with just a vouched session. Commenting might require a local account.
The trust group's permission level determines what's possible, and the
site's policy determines when to escalate from session to account.

**After account creation:**

Charlie has a real account on Alice's site. Alice doesn't know
Charlie's email on Bob's site, their password, or their user level.
She just knows Bob vouched for them. The vouching happened once. After
that, it's normal cookie-based auth — Charlie can log in directly on
future visits.

### What Alice Controls

- The permission level for Bob's trusted group (viewer, commenter)
- What requires a vouched session vs. a local account
- Whether to accept new vouched users from Bob's site
- Individual accounts once created — she can moderate, adjust
  permissions, or ban Charlie independently of the trust link
- Revoking the trust link stops new vouched sessions but doesn't
  delete existing accounts (those are her users now)

### What Bob Controls

- Which of his users can be vouched (all members? editors only?)
- Whether to vouch when Alice's site calls back
- His own user management — if he bans charlie, the next time
  Alice's site tries to verify, the voucher fails

### Engagement

Charlie with an account on Alice's site engages like any local user.
Comments live in Alice's `wp_comments`. Alice moderates them. Nothing
special — Charlie is a local user who happened to arrive through a
trust link.

The only difference from a normal account: Alice's admin shows that
Charlie's account was vouched by Bob's site. Provenance, not privilege.

### Revocation

Alice decides she no longer trusts Bob's site.

1. Alice removes the trust link from her admin panel
2. No new vouched accounts from Bob's site are accepted
3. Existing accounts created through the link remain — they're
   Alice's users now, she manages them individually
4. Alice can optionally bulk-remove all accounts from Bob's trusted
   group if she wants a clean break
5. Comments already left remain (they're Alice's data)

## Content Sovereignty

This system shifts the default from "public until scraped" to "shared
with intent."

- **Public posts** — anyone can read, including machines. The author
  chose this.
- **Shared posts** — only people with permission. Trusted site users,
  share token holders, direct grants. A scraper without a valid token
  gets nothing.
- **Private posts** — only the author.

The permission system isn't just about collaboration. It's about
sovereignty over your own words. You decide who reads what you write.
The trust network extends that decision across sites without surrendering
control.

In an age where every word you publish becomes training data for someone
else's product, the ability to share selectively — with real people,
through relationships you've chosen — matters.

## The Evolution of the Blogroll

The blogroll said "I like these sites." Trust links say "I vouch for
these sites." The blogroll was a list of URLs. Trust links are bilateral
agreements with mechanical consequences — real permissions, real access,
real identity verification.

The blogroll can still exist as the public face of these relationships.
Alice's sidebar shows "Trusted Sites" with links to Bob's blog and
Carol's blog. But behind each link is a trust agreement that lets their
communities interact.

## Open Questions

### Discovery

How do site owners find each other? For now, manual exchange is fine —
bloggers know other bloggers. But eventually: could a trust link include
"sites that my trusted sites trust"? Transitive trust, one level deep?
That's powerful but dangerous. Worth thinking about later.

### Scope granularity

Does Alice trust all of Bob's users equally? Or can she say "Bob's
editors can comment but Bob's subscribers can only view"? Bob controls
which of his users can be vouched. Alice controls the permission level
of the group they land in. But should Alice be able to differentiate
within that group? The tuple system supports it, but is it worth the
complexity?

### Content in feeds

If Alice trusts Bob's site, should Bob's shared posts appear in a
special feed for Alice's users? A "trusted network" feed alongside the
public feed? This starts approaching federation territory — tread
carefully.

### Callback protocol

The vouching callback is the core mechanic. It needs to be:

- Simple enough for two blogs to implement
- Secure enough that a forged callback can't create accounts
- Lightweight enough to not feel like enterprise SSO

Possible approaches: signed callback URLs with a shared secret from the
trust establishment, mutual TLS, or a simple challenge-response where
Alice's site asks Bob's site "do you vouch for this session?" and Bob's
site confirms. The right answer probably depends on how the key exchange
during trust establishment works.

### Account lifecycle

When Charlie creates an account on Alice's site through a voucher, that
account is Alice's. But should there be a periodic re-verification?
"Bob still vouches for Charlie" checked every N days? Or is the initial
voucher enough — once you're in, you're a local user and Alice manages
you from there?

### Compatibility

Could a non-Press site participate in the trust network? If the callback
protocol is simple enough, any blog platform could implement the voucher
endpoint. But standardizing too early risks designing the wrong thing.
Get it working between two Press sites first.

## What We Build Now

Nothing trust-specific. But the prerequisites matter:

1. **Tuple-based permissions** — trusted groups are just groups with
   tuples, same as any other group
2. **Groups** — the trusted site's users land in a group on the
   destination site
3. **Visibility levels** — `shared` visibility is the content category
   that trusted site users would unlock
4. **Account creation** — the registration flow that vouched users
   go through is the same registration flow as any user

## What We Don't Build Yet

- Trust link establishment UI
- Callback protocol and voucher verification
- Vouched account creation flow
- Trust group management in admin
- Blogroll integration (showing trust links publicly)
- Trust network discovery

These wait until the single-site experience is solid. The trust network
is a multiplayer feature built on top of a single-player foundation
that must work first.
