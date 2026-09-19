//go:build unit

package service

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testCommunityQRCode() string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nfixture"))
}

func TestNormalizeCommunityGroups(t *testing.T) {
	groups, err := NormalizeCommunityGroups([]CommunityGroup{{
		Name:        "  Official Group  ",
		GroupNumber: "  123456789  ",
		QRCodeImage: testCommunityQRCode(),
		JoinURL:     " https://example.com/join ",
	}})

	require.NoError(t, err)
	require.Equal(t, "Official Group", groups[0].Name)
	require.Equal(t, "123456789", groups[0].GroupNumber)
	require.Equal(t, "https://example.com/join", groups[0].JoinURL)
}

func TestNormalizeCommunityGroupsRejectsUnsafeOrInvalidContent(t *testing.T) {
	tests := []struct {
		name  string
		group CommunityGroup
	}{
		{name: "missing name", group: CommunityGroup{GroupNumber: "1"}},
		{name: "missing number", group: CommunityGroup{Name: "Group"}},
		{name: "unsafe link", group: CommunityGroup{Name: "Group", GroupNumber: "1", JoinURL: "javascript:alert(1)"}},
		{name: "wrong image signature", group: CommunityGroup{Name: "Group", GroupNumber: "1", QRCodeImage: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not png"))}},
		{name: "invalid image data", group: CommunityGroup{Name: "Group", GroupNumber: "1", QRCodeImage: "data:image/png;base64,%%%"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeCommunityGroups([]CommunityGroup{tt.group})
			require.Error(t, err)
		})
	}

	tooMany := make([]CommunityGroup, MaxCommunityGroups+1)
	for i := range tooMany {
		tooMany[i] = CommunityGroup{Name: "Group", GroupNumber: "1"}
	}
	_, err := NormalizeCommunityGroups(tooMany)
	require.ErrorContains(t, err, "too many")
}

func TestCommunityGroupsMarshalAndParseFailClosed(t *testing.T) {
	raw, err := MarshalCommunityGroups([]CommunityGroup{{Name: "Group", GroupNumber: "42"}})
	require.NoError(t, err)
	require.Equal(t, []CommunityGroup{{Name: "Group", GroupNumber: "42"}}, ParseCommunityGroups(raw))
	require.Empty(t, ParseCommunityGroups(`{"name":"not-an-array"}`))
	require.Empty(t, ParseCommunityGroups(`[{"name":"Group","group_number":"42","join_url":"file:///tmp/a"}]`))
	require.Empty(t, ParseCommunityGroups(strings.Repeat("x", 20)))
}
