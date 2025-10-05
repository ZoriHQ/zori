package typeid

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"github.com/speps/go-hashids/v2"
)

type EntityType interface {
	Prefix() string
	Salt() string
}

type TypeID[T EntityType] struct {
	value int64
	typ   T
}

func (id TypeID[T]) Prefix() string {
	return "default_prefix"
}

func (id TypeID[T]) Salt() string {
	return "default_salt"
}

var hashidsCache = make(map[string]*hashids.HashID)

func getHashID(prefix, salt string) *hashids.HashID {
	key := prefix + ":" + salt
	if hid, exists := hashidsCache[key]; exists {
		return hid
	}

	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = 8
	hd.Alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

	hid, _ := hashids.NewWithData(hd)
	hashidsCache[key] = hid
	return hid
}

func New[T EntityType](value int) TypeID[T] {
	var t T
	return TypeID[T]{value: int64(value), typ: t}
}

func Parse[T EntityType](s string) (TypeID[T], error) {
	var t T
	prefix := t.Prefix()

	if !strings.HasPrefix(s, prefix+"_") {
		return TypeID[T]{}, fmt.Errorf("invalid prefix for %s: %s", prefix, s)
	}

	encoded := strings.TrimPrefix(s, prefix+"_")
	hid := getHashID(prefix, t.Salt())

	decoded, err := hid.DecodeWithError(encoded)
	if err != nil {
		return TypeID[T]{}, fmt.Errorf("failed to decode %s: %w", s, err)
	}

	if len(decoded) == 0 {
		return TypeID[T]{}, fmt.Errorf("empty decode result for %s", s)
	}

	return TypeID[T]{value: int64(decoded[0]), typ: t}, nil
}

func (id TypeID[T]) String() string {
	prefix := id.typ.Prefix()
	hid := getHashID(prefix, id.typ.Salt())
	encoded, _ := hid.Encode([]int{int(id.value)})
	return prefix + "_" + encoded
}

func (id TypeID[T]) Int64() int64 {
	return id.value
}

func (id TypeID[T]) Int() int {
	return int(id.value)
}

func (id TypeID[T]) IsZero() bool {
	return id.value == 0
}

// --- Database interfaces ---

// Scan implements the sql.Scanner interface
func (id *TypeID[T]) Scan(value interface{}) error {
	if value == nil {
		id.value = 0
		return nil
	}

	switch v := value.(type) {
	case int64:
		id.value = v
	case int:
		id.value = int64(v)
	case string:
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot scan %T into TypeID: %w", value, err)
		}
		id.value = val
	case []byte:
		val, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("cannot scan %T into TypeID: %w", value, err)
		}
		id.value = val
	default:
		return fmt.Errorf("cannot scan %T into TypeID", value)
	}

	return nil
}

// Value implements the driver.Valuer interface
func (id TypeID[T]) Value() (driver.Value, error) {
	return id.value, nil
}

// --- JSON marshaling ---

func (id TypeID[T]) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.String() + `"`), nil
}

func (id *TypeID[T]) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	parsed, err := Parse[T](s)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type ProjectID struct{}

func (ProjectID) Prefix() string { return "zpr" }
func (ProjectID) Salt() string   { return "zori-project-salt-key" }

// --- Helper functions ---

// NewProjectID creates a new ProjectID TypeID
func NewProjectID(value int) TypeID[ProjectID] {
	return New[ProjectID](value)
}

type KeywordID struct{}

func (KeywordID) Prefix() string { return "kw" }
func (KeywordID) Salt() string   { return "kw-salt-secret-key" }
func NewKeywordID(value int) TypeID[KeywordID] {
	return New[KeywordID](value)
}

type MatchID struct{}

func (MatchID) Prefix() string { return "mch" }
func (MatchID) Salt() string   { return "match-salt-secret-key" }

func NewMatchID(value int) TypeID[MatchID] {
	return New[MatchID](value)
}

type ReplyID struct{}

func (ReplyID) Prefix() string { return "rep" }
func (ReplyID) Salt() string   { return "reply-salt-secret-key" }

func NewReplyID(value int) TypeID[ReplyID] {
	return New[ReplyID](value)
}

type SubredditID struct{}

func (SubredditID) Prefix() string { return "subr" }
func (SubredditID) Salt() string   { return "subr-salt-secret-key" }
func NewSubredditID(value int) TypeID[SubredditID] {
	return New[SubredditID](value)
}

type BookmarkID struct{}

func (BookmarkID) Prefix() string { return "bkm" }
func (BookmarkID) Salt() string   { return "bkm-salt-secret-key" }
func NewBookmarkID(value int) TypeID[BookmarkID] {
	return New[BookmarkID](value)
}

// ParseInt parses a string as an integer
func ParseInt(s string) (int, error) {
	value, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %s", s)
	}
	return value, nil
}
