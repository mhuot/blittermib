/*
 * Copyright 2026 Ronny Trommer <ronny@no42.org>
 * SPDX-License-Identifier: MIT
 */

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "x-real-ip wins",
			remoteAddr: "10.0.0.1:5000",
			headers:    map[string]string{"X-Real-IP": "203.0.113.7"},
			want:       "203.0.113.7",
		},
		{
			name:       "first x-forwarded-for hop when no real-ip",
			remoteAddr: "10.0.0.1:5000",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.9, 10.0.0.1"},
			want:       "198.51.100.9",
		},
		{
			name:       "remote addr port stripped when unproxied",
			remoteAddr: "192.0.2.44:53314",
			want:       "192.0.2.44",
		},
		{
			name:       "remote addr returned verbatim when unparseable",
			remoteAddr: "unix-socket",
			want:       "unix-socket",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			if got := clientIP(r); got != tc.want {
				t.Errorf("clientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}
