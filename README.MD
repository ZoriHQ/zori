# zori.

revenue → attribution → clarity.

---

## the problem?
you know your traffic.
you don't know your money.
the gap between visitor and customer is dark.

## the answer.
zori watches first touch.
remembers everything.
connects dots when stripe speaks.

---

## what lives inside

```
visitor arrives (utm_source=magic)
     ↓
zori remembers
     ↓
[... time passes ...]
     ↓
visitor → customer ($$$)
     ↓
"ah, magic brought money"
```

**first touch.**
where they came from first
matters most

**every click.**
buttons. links. intentions.
classified. understood.

**revenue flows.**
mrr. ltv. conversion time.
truth in numbers.

---

## architecture (ascii version)

```
┌──────────┐     ┌──────────┐
│ browser  │────▶│ingestion │──┐
└──────────┘     └──────────┘  │
                                ▼
┌──────────┐     ┌──────────┐  NATS
│  stripe  │────▶│processor │◀─┘
└──────────┘     └──────────┘
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   ┌─────────┐  ┌──────────┐  ┌─────────┐
   │postgres │  │clickhouse│  │  redis  │
   └─────────┘  └──────────┘  └─────────┘
    (who you)    (what did)    (cache)
```

---

## quickstart recipe

**ingredients:**
- go 1.24+ (fresh)
- postgres 15+ (relational truth)
- clickhouse 24+ (time series speed)
- redis 7+ (memory)
- nats (message river)

**preparation:**

1. clone the repository
```bash
git clone https://github.com/ZoriHQ/zori.git && cd zori
```

2. season environment
```bash
cp example.env .env
# add your secrets to taste
```

3. rise the database
```bash
task migrate:up
```

4. serve hot
```bash
task server      # port 1323
task ingestion   # separate process port 1324
```

done. tracking begins.

---

## modes of being

**oss mode**
`ZORI_IS_OSS=true`
single tenant. your auth. your stripe key.
freedom.

**cloud mode**
multi-tenant. clerk auth. stripe oauth.
scale.

---

## questions → answers

**q: why track first touch only?**
a: first impression brought them. credit where due.

**q: visitor_id vs session_id?**
a: visitor persists forever. session = one sitting.

**q: why clickhouse?**
a: billions of events. millisecond aggregations. compression.

**q: what gets classified?**
a: every click. button/link/form/nav. patterns emerge.

---

## environment whispers

```bash
POSTGRES_URL=postgres://localhost:5432/zori
CLICKHOUSE_URL=http://localhost:8123
REDIS_ADDRS=localhost:6379
NATS_STREAM_URL=nats://localhost:4222

# choose your path
ZORI_IS_OSS=true           # or
CLERK_SECRET_KEY=sk_test_  # cloud
```

---

## dev commands

```bash
task test          # prove it works
task docs          # swagger at :8080/swagger
```

---

## license

apache 2.0

---

built with go.
sees through clickhouse.
remembers in postgres.
messages through nats.

track money. not just traffic.
