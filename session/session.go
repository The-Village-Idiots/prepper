// Copyright (C) 2023 - Ethan Marshall
// Written for A-Level Computer Science Project 2024

// Package session implements a session store for server-side sessions based on
// HTTP cookies. The session store is a thread safe map between a 32-character
// ASCII token and details which are stored non-persistently.
package session

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Session token format definitions.
const (
	// The name of the cookie which stores the session token.
	TokenCookieName = "SESSID"
	// The length of time between which sessions will be invalidated.
	MaxSessionLength = 24 * time.Hour

	// Minimum printing ASCII value.
	MinTokenByte = 0x30
	// Maximum printing ASCII value.
	MaxTokenByte = 0x7E

	// Characters forbidden by the Set-Cookie standard.
	ForbiddenCharacters = "\"\\,;"

	// Maximum number of retries before panicking on failure.
	maxTokenRetries = 10
)

var (
	minTokenBig = big.NewInt(MinTokenByte)
	maxTokenBig = big.NewInt(MaxTokenByte - MinTokenByte)
)

// Decoding and encoding errors for session tokens.
var (
	// A byte in the session token was out of range.
	ErrByteRange = fmt.Errorf("byte in token out of range")
	// The session token was the wrong length.
	ErrLength = fmt.Errorf("token wrong length")

	// Sentinel for encoding loop.
	// Do not use otherwise.
	errAgain = fmt.Errorf("again")
)

// A Session is a single instance of a user session. Sessions may never contain
// references to thread unsafe data and should not store particularly large
// objects. Sessions are always passed by reference.
type Session struct {
	// The associated session token.
	Token Token
	// If SignedIn is true, UserID points to a valid user ID
	SignedIn bool
	// UserID is the ID of the currently signed in user
	UserID uint

	// Timestamp of the creation time of this session.
	Created time.Time

	// Convenience: allows short hand saving
	store *Store
}

func (s *Session) SignIn(id uint) {
	s.UserID = id
	s.SignedIn = true
}

// Update is a convenience method to update the currently returned session.
func (s *Session) Update() {
	if s.store == nil {
		return
	}

	s.store.Update(s)
}

// Session store is a thread safe map between a Session instance and a 32-byte
// session token.
type Store struct {
	*sync.RWMutex
	s map[Token]Session
}

// NewStore allocates and returns a new session store with a blank session map.
func NewStore() Store {
	return Store{
		new(sync.RWMutex),
		make(map[Token]Session),
	}
}

// Lookup finds and returns the session associated with the given token, if one
// exists.
func (s *Store) Lookup(t Token) (Session, bool) {
	s.RLock()
	defer s.RUnlock()

	sess, ok := s.s[t]
	return sess, ok
}

// Exists returns the second return value from s.Lookup, which indicates if a
// sessions is associated with the given token.
func (s *Store) Exists(t Token) bool {
	_, ret := s.Lookup(t)
	return ret
}

// New creates a new blank session, adding it to the sessions map and
// generating a new, guaranteed unique token for it.
func (s *Store) New() Session {
	tok := NewToken()
	// Retry with new token until a unique one is found
	for s.Exists(tok) {
		tok = NewToken()
	}

	sess := Session{
		Token:   tok,
		Created: time.Now(),
		store:   s,
	}

	s.Lock()
	defer s.Unlock()
	s.s[tok] = sess
	return sess
}

func (s *Store) doStart(ctx *gin.Context) Session {
	sess := s.New()
	c := &http.Cookie{
		Name:   TokenCookieName,
		Value:  sess.Token.String(),
		MaxAge: int(MaxSessionLength.Seconds()),
		Path:   "/",
	}

	http.SetCookie(ctx.Writer, c)
	return sess
}

// Start is called at the beginning of any request which requires access to a
// session. If a session exists, it is retrieved. Else, a blank session is
// created and returned, with the necessary cookie having been set.
func (s *Store) Start(c *gin.Context) Session {
	rtok, err := c.Cookie(TokenCookieName)
	if err != nil {
		// No given cookie. Give one back instead.
		return s.doStart(c)
	}

	tok, err := ParseToken(rtok)
	if err != nil {
		// Bad token. Give back a correct one.
		return s.doStart(c)
	}

	sess, found := s.Lookup(tok)
	if !found {
		// Token that does not exist. Give back one that does.
		return s.doStart(c)
	}

	// Just in case of incorrect initialisation
	if sess.store == nil {
		sess.store = s
	}

	return sess
}

// Update saves any changes made to sess into the session map for other
// requests to use. If sess does not yet exist, it is created.
func (s *Store) Update(sess *Session) {
	s.Lock()
	defer s.Unlock()

	s.s[sess.Token] = *sess
}

// A Token is a 32 byte string which must consist of exclusively printable
// ASCII characters.
type Token [32]byte

func (t Token) String() string {
	return string(t[:])
}

// NewToken generates a new token, which is an ASCII string of 32-bytes in
// length. It is generated using a cryptographic RNG source.
func NewToken() Token {
	tok := Token{}

	for i := 0; i < len(tok); i++ {
		err := errAgain
		count := 0

		var n *big.Int
		for err != nil {
			if count > maxTokenRetries {
				panic("new token: reached maximum generation retries")
			}

			n, err = rand.Int(rand.Reader, maxTokenBig)
			if err != nil {
				continue
			}

			count++
		}

		n = n.Add(n, minTokenBig)

		trunc := n.Int64()
		ins := byte(trunc)

		// Please see net/http's sanitizeCookieValue function for more
		// info on why this is required.
		for i, r := range ForbiddenCharacters {
			if ins == byte(r) {
				ins = byte('0') + byte(i)
			}
		}

		tok[i] = ins
	}

	return tok
}

// ParseToken parses a session token from the byte string in, returning the
// first error encountered and what was decoded so far.
func ParseToken(in string) (Token, error) {
	tok := Token{}
	if len(in) != len(tok) {
		return tok, fmt.Errorf("parse token %s: %w (%d, expect %d)", in, ErrLength, len(in), len(tok))
	}

	for i, r := range in {
		if r < MinTokenByte || r > MaxTokenByte {
			return tok, fmt.Errorf("parse token %s: %w at index %d", in, ErrByteRange, i)
		}

		tok[i] = byte(r)
	}

	return tok, nil
}
