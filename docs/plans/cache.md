# Cache

## Problem

Press will have many things worth caching — permission checks, options,
user metadata, template fragments. Right now everything hits the DB on
every request. We need a caching layer that:

- Starts simple (in-memory)
- Can swap to Valkey/Redis without changing application code
- Is available anywhere via context, not threaded through constructors
- Lets each subsystem (permissions, options, etc.) own its own cache
  logic internally — callers don't think about caching

## Two layers

### Store (backend interface)

The low-level backend. Deals in bytes. Swappable.

```go
package cache

type Store interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    Flush(ctx context.Context) error
}
```

First implementation: `MemoryStore` — a `sync.RWMutex` + `map[string]entry`
with TTL expiry. No external dependencies.

Future: `ValkeyStore`, `RedisStore` — same interface, different backend.
The application configures which store to use via `.env`
(`CACHE_DRIVER=memory|valkey`).

### Cache (application-level)

The thing application code interacts with. Handles serialization,
key prefixing, and the `Remember` pattern.

```go
type Cache struct {
    store  Store
    prefix string
}

func New(store Store) *Cache
func NewNoop() *Cache  // no-op cache for tests/CLI — everything passes through
```

Generic functions for type-safe access:

```go
func Get[T any](ctx context.Context, c *Cache, key string) (T, bool, error)
func Set[T any](ctx context.Context, c *Cache, key string, value T, ttl time.Duration) error
func Remember[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, fn func() (T, error)) (T, error)
func Forget(ctx context.Context, c *Cache, keys ...string) error
```

When `c` is nil (no cache in context), `Remember` calls the function
directly and returns the result. No panics, no `if cache != nil`
guards in application code.

## Context

The cache is created at server startup and injected into every request
context via middleware.

```go
// Server startup
appCache := cache.New(cache.NewMemoryStore())

// Middleware
func WithCache(c *Cache) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := cache.WithContext(r.Context(), c)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Anywhere downstream
c := cache.FromContext(ctx)  // nil-safe — Remember handles nil
```

CLI commands and tests don't set up cache middleware. `FromContext`
returns nil, `Remember` passes through. No special handling needed.

## Serialization

JSON. It's debuggable, adequate for the payload sizes we're caching
(permission tuples, option values, small structs), and works across
every backend. If profiling shows serialization is a bottleneck, swap
to gob or msgpack — the Store interface doesn't care.

## Invalidation

TTL + explicit delete on mutation. No tags for now.

- **TTL** — every cached value has a TTL. Permission group lookups
  might be 5 minutes. Options might be longer. Each subsystem chooses
  its own TTLs.
- **Explicit delete** — when a mutation happens (group membership
  change, option update), the code that performs the mutation deletes
  the relevant cache keys. The subsystem owns its key format, so it
  knows what to delete.

Tags (cache groups that can be flushed together) are useful but add
complexity to the Store interface. Add later if invalidation gets
unwieldy.

## Usage pattern

Any subsystem that wants caching wraps its own DB/compute calls with
`cache.Remember` internally. The caller never knows caching exists.
The subsystem owns its key format, TTLs, and invalidation.

Likely consumers beyond permissions:

- **Options** — `wp_options` with `autoload = 'yes'` are read on every
  request. Cache the whole autoload set as one key.
- **Query results** — post listings, term lookups, user metadata. Any
  read-heavy query that doesn't change often. Key by the query
  parameters, invalidate on write.
- **Template fragments** — rendered HTML partials (sidebar, blogroll,
  recent posts). Invalidate when underlying data changes.
- **User sessions/metadata** — user display names, email, nicename.
  Read every request, written rarely.

The pattern is always the same:

```go
func (r *PostRepository) ListPublished(ctx context.Context, page int) ([]Post, error) {
    key := fmt.Sprintf("posts:published:%d", page)
    return cache.Remember(ctx, cache.FromContext(ctx), key, 10*time.Minute, func() ([]Post, error) {
        return r.queryPublished(ctx, page)
    })
}
```

## Permissions (first use case)

The permission checker wraps its own DB calls with cache logic
internally. Callers just call `Can()` — they don't know caching exists.

### What to cache

| Data | Key | TTL | Invalidate on |
|------|-----|-----|---------------|
| User's groups | `perm:groups:{userID}` | 5m | Membership tuple create/delete |
| Group grants for object | `perm:grants:{objectType}:{objectID}` | 5m | Group grant tuple create/delete |

The full `Can()` result is not cached initially. Caching the two
underlying queries cuts DB hits substantially without complex
invalidation. If needed later, cache `Can()` results with short TTL.

### Inside the checker

```go
func (c *checker) getUserGroups(ctx context.Context, userID int64) ([]string, error) {
    key := fmt.Sprintf("perm:groups:%d", userID)
    return cache.Remember(ctx, cache.FromContext(ctx), key, 5*time.Minute, func() ([]string, error) {
        return c.store.GetUserGroups(ctx, userID)
    })
}
```

### Invalidation

The store mutation methods (`CreateTuple`, `DeleteTuple`) are the
natural place. After a successful write, delete the relevant keys:

```go
func (s *Store) CreateTuple(ctx context.Context, t *Tuple) error {
    // ... insert into DB ...

    // Invalidate cache.
    if c := cache.FromContext(ctx); c != nil {
        if t.Relation == Member && t.ObjectType == "group" {
            cache.Forget(ctx, c, fmt.Sprintf("perm:groups:%s", t.SubjectID))
        }
        if t.SubjectType == "group" {
            cache.Forget(ctx, c, fmt.Sprintf("perm:grants:%s:%s", t.ObjectType, t.ObjectID))
        }
    }
    return nil
}
```

## Memory store implementation

```go
type entry struct {
    data      []byte
    expiresAt time.Time
}

type MemoryStore struct {
    mu      sync.RWMutex
    entries map[string]entry
}
```

Expired entries are cleaned up lazily on access and periodically by a
background goroutine (e.g., every 60 seconds). No external dependencies.

## File structure

```
internal/cache/
    store.go       — Store interface
    memory.go      — MemoryStore implementation
    cache.go       — Cache struct, generic functions, context helpers
```

## Future

- Valkey/Redis store implementation
- Tags for group invalidation
- Cache stats/metrics (hit rate, miss rate)
- Per-request deduplication (dataloader pattern for batch queries)
