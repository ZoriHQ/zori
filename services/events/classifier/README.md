# Event Click Classification System

## Overview

The event click classification system automatically analyzes click events to answer questions like:
- Did the user click on a button or a link?
- Was it a CTA (Call-To-Action)?
- What type of element was clicked?
- If it was a link, where does it lead?
- Is it an external link or download?

## Architecture

### Components

1. **Classifier Package** (`services/events/classifier/`)
   - `types.go`: Defines element types and categories
   - `classifier.go`: Core classification logic
   - `classifier_test.go`: Unit tests

2. **Processing Stage** (`services/events/services/stage_click_classification.go`)
   - Integrates with event processor pipeline
   - Runs after other enrichment stages

3. **Database Schema** (ClickHouse)
   - Migration: `20251028000001_add_click_classification_fields.sql`
   - New columns in `events` table

## Element Types

The classifier categorizes clicks into these element types:

- `button` - Button elements
- `link` - Anchor (`<a>`) elements
- `input` - Input fields
- `text_element` - Text containers (div, span, p, headings)
- `image` - Image elements
- `video` - Video elements
- `navigation` - Navigation menu items
- `form_control` - Form controls (select, textarea, checkboxes)
- `other` - Uncategorized elements

## Element Categories

Higher-level groupings:

- `cta` - Call-to-action elements (sign up, buy, download, etc.)
- `navigation` - Links and nav menus
- `content` - Content elements
- `form` - Form inputs and controls
- `media` - Images and videos
- `other` - Uncategorized

## Classification Fields

After processing, each click event includes:

| Field | Type | Description |
|-------|------|-------------|
| `click_element_type` | String | Specific element type |
| `click_element_category` | String | High-level category |
| `is_cta_click` | Boolean | Whether it's a CTA |
| `link_destination` | String | URL for link clicks |
| `is_external_link` | Boolean | External domain link |
| `is_download_link` | Boolean | Link to downloadable file |

## Usage Examples

### SQL Queries

**Count CTA clicks:**
```sql
SELECT COUNT(*)
FROM events
WHERE is_cta_click = true
  AND project_id = ?
  AND client_timestamp_utc >= ?
```

**Analyze clicks by element type:**
```sql
SELECT
  click_element_type,
  COUNT(*) as click_count
FROM events
WHERE click_element_type IS NOT NULL
  AND project_id = ?
GROUP BY click_element_type
ORDER BY click_count DESC
```

**Find most clicked CTAs:**
```sql
SELECT
  click_element_text,
  COUNT(*) as clicks
FROM events
WHERE is_cta_click = true
  AND click_element_text IS NOT NULL
  AND project_id = ?
GROUP BY click_element_text
ORDER BY clicks DESC
LIMIT 10
```

**External vs internal link clicks:**
```sql
SELECT
  is_external_link,
  COUNT(*) as link_clicks
FROM events
WHERE click_element_type = 'link'
  AND project_id = ?
GROUP BY is_external_link
```

**Button clicks by category:**
```sql
SELECT
  click_element_category,
  COUNT(*) as button_clicks
FROM events
WHERE click_element_type = 'button'
  AND project_id = ?
GROUP BY click_element_category
ORDER BY button_clicks DESC
```

### Analytics Integration

The classification fields are automatically populated during event processing and can be used in:

1. **Funnel Analysis**: Track CTA clicks through conversion funnels
2. **A/B Testing**: Compare performance of different button types
3. **User Behavior**: Understand what users click most
4. **Conversion Tracking**: Measure CTA effectiveness
5. **Navigation Analysis**: See how users navigate your site

## CTA Detection

The system automatically detects CTAs based on:

### Text Patterns (case-insensitive)
- Sign up / Signup / Register
- Get Started / Start Free
- Buy Now / Purchase / Add to Cart / Checkout
- Subscribe / Join
- Download / Try Free / Free Trial
- Contact / Request / Schedule / Book
- Learn More / Get / Claim / Unlock / Upgrade
- Submit / Send / Apply / Enroll

### Selector Patterns
- `.cta`
- `.call-to-action`
- `.btn-primary`
- `.btn-cta`

## How It Works

1. **Event Ingestion**: Client sends click event with element tag, selector, and text
2. **Processing Pipeline**: Event goes through enrichment stages
3. **Classification Stage**: Analyzer examines the click element:
   - Identifies element type from tag and selector
   - Checks text content for CTA patterns
   - Extracts metadata (hrefs, alt text, etc.)
   - Determines high-level category
4. **Storage**: Classification fields stored in ClickHouse
5. **Querying**: Analytics queries use classification fields for insights

## Extending the System

### Adding New CTA Patterns

Edit `services/events/classifier/classifier.go`:

```go
var ctaPatterns = []string{
    // ... existing patterns
    "your new pattern",
}
```

### Adding New Element Types

1. Add to `ElementType` enum in `types.go`
2. Update `classifyElementType()` in `classifier.go`
3. Add test cases in `classifier_test.go`

### Custom Classification Logic

Modify `Classify()` method in `classifier.go` to add custom rules:

```go
func (c *Classifier) Classify(tag, selector, text string) *Classification {
    // Your custom logic here
}
```

## Testing

Run tests:
```bash
go test ./services/events/classifier/...
```

## Migration

Run the migration to add classification fields:
```bash
task migrate:up
# or
GOOSE_DRIVER=clickhouse GOOSE_DBSTRING="..." goose -dir internal/storage/clickhouse/migrations up
```

## Performance

- Classification happens during event processing (async)
- Indexed fields enable fast queries
- Bloom filter indexes on classification columns
- No runtime overhead for analytics queries
