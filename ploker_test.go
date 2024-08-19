package ploker

import "testing"

func TestGetSessionID(t *testing.T) {

	for _, tt := range []struct {
		surl   string
		expect string
		error  bool
	}{
		{
			surl:   "http://www.whatever.com/session/abcdef",
			expect: "abcdef",
		},
		{
			surl:  "http://www.whatever.com/session/abcdef/",
			error: true,
		},
		{
			surl:  "http://www.whatever.com/session",
			error: true,
		},
		{
			surl:  "http://www.whatever.com/session/",
			error: true,
		},
		{
			surl:  "http://www.whatever.com/",
			error: true,
		},
		{
			surl:  "http://www.whatever.com/somethingelse",
			error: true,
		},
	} {

		t.Run(tt.expect, func(t *testing.T) {
			s, err := GetSessionID(tt.surl)
			if tt.error {
				if err == nil {
					t.Fatalf("Expected error, but got session id: %v", s)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if s != tt.expect {
				t.Fatalf("Expected session id == \"abcdef\" but got %v", s)
			}
		})
	}
}
