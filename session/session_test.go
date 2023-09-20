// Copyright (C) 2023 - Ethan Marshall
// Written for A-Level Computer Science Project 2024

package session

import (
	"errors"
	"strings"
	"testing"
)

func TestNewToken(t *testing.T) {
	tok := NewToken()

	for i, b := range tok {
		if b < MinTokenByte || b > MaxTokenByte {
			t.Errorf("%s: token byte out of range @ %d 0x%x '%c'", tok, i, b, b)
		}
	}

	if len(tok) != 32 {
		t.Errorf("token wrong length (expect 32, got %d)", len(tok))
	}

	if strings.ContainsAny(tok.String(), ForbiddenCharacters) {
		t.Error("token contains blacklisted characters (", tok.String(), ")")
	}

	t.Log(tok)
}

func stringToBytes(s string) [32]byte {
	if len(s) > 32 {
		panic("use of stringToBytes with string len() > 32")
	}

	ret := [32]byte{}
	// Note: intentionally avoiding range loop to avoid UTF-8 decoding
	for i := 0; i < len(s); i++ {
		ret[i] = s[i]
	}

	return ret
}

func TestParseToken(t *testing.T) {
	cases := []struct {
		Src       string
		Expect    *Token
		Error     bool
		ErrorText error
	}{
		// Success cases
		{"lKXstaw^kJzt|]T_L_HT^zM}JI{u}hYH", nil, false, nil},

		// Failure cases
		// Fails due to length requirement (lower bound)
		{"", &Token{}, true, ErrLength},
		// Fails due to length requirement (upper bound)
		{strings.Repeat("a", 40), &Token{}, true, ErrLength},
		// Fails as '£' is out of ASCII range
		{"£" + strings.Repeat("!", 30), nil, true, ErrByteRange},
	}

	for _, c := range cases {
		t.Run(c.Src, func(t *testing.T) {
			exp := c.Expect
			if exp == nil {
				v := Token(stringToBytes(c.Src))
				exp = &v
			}

			tok, err := ParseToken(c.Src)
			if c.Error {
				if err == nil {
					t.Error("expected error, got successful parse")
				}

				if !errors.Is(err, c.ErrorText) {
					t.Error("wrong error returned, expected", c.ErrorText, "got", err)
				}
			} else {
				if err != nil {
					t.Error("unexpected error, expected successful parse")
				}

				if tok != *exp {
					t.Errorf("bad result from parse\nexpect: %v\ngot: %v", c.Expect, tok)
				}
			}
		})
	}

}
