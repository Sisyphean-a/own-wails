package profileutil

import "testing"

func TestNormalizeRelativePath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "windows drive", in: `C:\\Users\\me\\.gitconfig`, want: `Users/me/.gitconfig`},
		{name: "home template", in: `{{HOME}}/.ssh/config`, want: `.ssh/config`},
		{name: "leading slash", in: `/etc/hosts`, want: `etc/hosts`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeRelativePath(tc.in)
			if got != tc.want {
				t.Fatalf("NormalizeRelativePath() = %q, want %q", got, tc.want)
			}
		})
	}
}
