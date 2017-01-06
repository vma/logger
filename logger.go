// Copyright 2017 Vallimamod Abdullah <vma@vallimamod.org>.
// The original code is borrowed from https://github.com/gorilla/handlers/blob/master/handlers.go
// and is Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logger

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/pressly/chi/middleware"
)

// CombinedLogger and CommonLogger are middlewares that log to stderr all
// http requests and response in Apache CommonLog and CombinedLog formats.
// They also log the answer delay in ms.

// These loggers are directly ported from gorilla handlers
// https://github.com/gorilla/handlers

// CombinedLogger returns a middleware that logs HTTP requests to `out` Writer
// in combined log format
func CombinedLogger(out io.Writer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			t1 := time.Now()
			w2 := middleware.NewWrapResponseWriter(w)
			next.ServeHTTP(w2, r)
			t2 := time.Now()
			writeCombinedLog(out, r, t1, w2.Status(), w2.BytesWritten(), t2.Sub(t1))
		}
		return http.HandlerFunc(fn)
	}
}

// CommonLogger returns a middleware that logs HTTP requests to `out` Writer
// in common log format
func CommonLogger(out io.Writer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			t1 := time.Now()
			w2 := middleware.NewWrapResponseWriter(w)
			next.ServeHTTP(w2, r)
			t2 := time.Now()
			writeCommonLog(out, r, t1, w2.Status(), w2.BytesWritten(), t2.Sub(t1))
		}
		return http.HandlerFunc(fn)
	}
}

const lowerhex = "0123456789abcdef"

func appendQuoted(buf []byte, s string) []byte {
	var runeTmp [utf8.UTFMax]byte
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		if r == rune('"') || r == '\\' { // always backslashed
			buf = append(buf, '\\')
			buf = append(buf, byte(r))
			continue
		}
		if strconv.IsPrint(r) {
			n := utf8.EncodeRune(runeTmp[:], r)
			buf = append(buf, runeTmp[:n]...)
			continue
		}
		switch r {
		case '\a':
			buf = append(buf, `\a`...)
		case '\b':
			buf = append(buf, `\b`...)
		case '\f':
			buf = append(buf, `\f`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		case '\v':
			buf = append(buf, `\v`...)
		default:
			switch {
			case r < ' ':
				buf = append(buf, `\x`...)
				buf = append(buf, lowerhex[s[0]>>4])
				buf = append(buf, lowerhex[s[0]&0xF])
			case r > utf8.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				buf = append(buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			default:
				buf = append(buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	return buf
}

// prettyDuration pretty-prints page rendering delay in ms
func prettyDuration(dur time.Duration) string {
	ms := float64(dur.Nanoseconds()) / float64(1e6)
	return fmt.Sprintf("%.3fms", ms)
}

// buildCommonLogLine builds a log entry for req in Apache Common Log Format.
// ts is the timestamp with which the entry should be logged.
// status and size are used to provide the response HTTP status and size.
func buildCommonLogLine(req *http.Request, ts time.Time, status int, size int, delay time.Duration) []byte {
	username := "-"
	url := *req.URL
	if url.User != nil {
		if name := url.User.Username(); name != "" {
			username = name
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		host = req.RemoteAddr
	}

	uri := req.RequestURI

	// Requests using the CONNECT method over HTTP/2.0 must use
	// the authority field (aka r.Host) to identify the target.
	// Refer: https://httpwg.github.io/specs/rfc7540.html#CONNECT
	if req.ProtoMajor == 2 && req.Method == "CONNECT" {
		uri = req.Host
	}
	if uri == "" {
		uri = url.RequestURI()
	}

	buf := make([]byte, 0, 3*(len(host)+len(username)+len(req.Method)+len(uri)+len(req.Proto)+50)/2)
	buf = append(buf, host...)
	buf = append(buf, " - "...)
	buf = append(buf, username...)
	buf = append(buf, " ["...)
	buf = append(buf, ts.Format("02/Jan/2006:15:04:05 -0700")...)
	buf = append(buf, `] "`...)
	buf = append(buf, req.Method...)
	buf = append(buf, " "...)
	buf = appendQuoted(buf, uri)
	buf = append(buf, " "...)
	buf = append(buf, req.Proto...)
	buf = append(buf, `" `...)
	buf = append(buf, strconv.Itoa(status)...)
	buf = append(buf, " "...)
	buf = append(buf, strconv.Itoa(size)...)
	buf = append(buf, " "...)
	buf = append(buf, prettyDuration(delay)...)
	return buf
}

// writeCommonLog writes a log entry for req to w in Apache Common Log Format.
// ts is the timestamp with which the entry should be logged.
// status, size and delay are used to provide the response HTTP status, size and delay.
func writeCommonLog(w io.Writer, req *http.Request, ts time.Time, status, size int, delay time.Duration) {
	buf := buildCommonLogLine(req, ts, status, size, delay)
	buf = append(buf, '\n')
	w.Write(buf)
}

// writeCombinedLog writes a log entry for req to w in Apache Combined Log Format.
// ts is the timestamp with which the entry should be logged.
// status, size and delay are used to provide the response HTTP status, size and delay.
func writeCombinedLog(w io.Writer, req *http.Request, ts time.Time, status, size int, delay time.Duration) {
	buf := buildCommonLogLine(req, ts, status, size, delay)
	buf = append(buf, ` "`...)
	buf = appendQuoted(buf, req.Referer())
	buf = append(buf, `" "`...)
	buf = appendQuoted(buf, req.UserAgent())
	buf = append(buf, '"', '\n')
	w.Write(buf)
}
