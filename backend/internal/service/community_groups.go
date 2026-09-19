package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	MaxCommunityGroups          = 12
	MaxCommunityGroupNameRunes  = 64
	MaxCommunityGroupNoRunes    = 80
	MaxCommunityGroupJoinURLLen = 2048
	MaxCommunityGroupQRCodeSize = 300 * 1024
)

var allowedCommunityQRCodePrefixes = []string{
	"data:image/png;base64,",
	"data:image/jpeg;base64,",
	"data:image/jpg;base64,",
	"data:image/webp;base64,",
}

// CommunityGroup is an administrator-configured user community entry. QR
// images remain in the settings store and are fetched only when a signed-in
// user opens the community dialog.
type CommunityGroup struct {
	Name        string `json:"name"`
	GroupNumber string `json:"group_number"`
	QRCodeImage string `json:"qr_code_image"`
	JoinURL     string `json:"join_url"`
}

func NormalizeCommunityGroups(groups []CommunityGroup) ([]CommunityGroup, error) {
	if len(groups) > MaxCommunityGroups {
		return nil, fmt.Errorf("too many community groups (max %d)", MaxCommunityGroups)
	}
	normalized := make([]CommunityGroup, 0, len(groups))
	for i, group := range groups {
		group.Name = strings.TrimSpace(group.Name)
		group.GroupNumber = strings.TrimSpace(group.GroupNumber)
		group.QRCodeImage = strings.TrimSpace(group.QRCodeImage)
		group.JoinURL = strings.TrimSpace(group.JoinURL)
		if group.Name == "" {
			return nil, fmt.Errorf("community group %d name is required", i+1)
		}
		if utf8.RuneCountInString(group.Name) > MaxCommunityGroupNameRunes {
			return nil, fmt.Errorf("community group %d name is too long", i+1)
		}
		if group.GroupNumber == "" {
			return nil, fmt.Errorf("community group %d number is required", i+1)
		}
		if utf8.RuneCountInString(group.GroupNumber) > MaxCommunityGroupNoRunes {
			return nil, fmt.Errorf("community group %d number is too long", i+1)
		}
		if group.JoinURL != "" {
			if len(group.JoinURL) > MaxCommunityGroupJoinURLLen || config.ValidateAbsoluteHTTPURL(group.JoinURL) != nil {
				return nil, fmt.Errorf("community group %d link must be an absolute http(s) URL", i+1)
			}
		}
		if group.QRCodeImage != "" {
			if err := validateCommunityQRCode(group.QRCodeImage); err != nil {
				return nil, fmt.Errorf("community group %d QR code: %w", i+1, err)
			}
		}
		normalized = append(normalized, group)
	}
	return normalized, nil
}

func validateCommunityQRCode(value string) error {
	encoded := ""
	mimeType := ""
	for _, prefix := range allowedCommunityQRCodePrefixes {
		if strings.HasPrefix(value, prefix) {
			encoded = strings.TrimPrefix(value, prefix)
			mimeType = prefix
			break
		}
	}
	if encoded == "" {
		return errors.New("must be a PNG, JPEG, or WebP data image")
	}
	if base64.StdEncoding.DecodedLen(len(encoded)) > MaxCommunityGroupQRCodeSize {
		return fmt.Errorf("image exceeds %dKB", MaxCommunityGroupQRCodeSize/1024)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return errors.New("contains invalid base64 data")
	}
	if len(decoded) == 0 || len(decoded) > MaxCommunityGroupQRCodeSize {
		return fmt.Errorf("image must be between 1 byte and %dKB", MaxCommunityGroupQRCodeSize/1024)
	}
	validSignature := false
	switch mimeType {
	case "data:image/png;base64,":
		validSignature = bytes.HasPrefix(decoded, []byte("\x89PNG\r\n\x1a\n"))
	case "data:image/jpeg;base64,", "data:image/jpg;base64,":
		validSignature = len(decoded) >= 3 && decoded[0] == 0xff && decoded[1] == 0xd8 && decoded[2] == 0xff
	case "data:image/webp;base64,":
		validSignature = len(decoded) >= 12 && string(decoded[:4]) == "RIFF" && string(decoded[8:12]) == "WEBP"
	}
	if !validSignature {
		return errors.New("file signature does not match its image type")
	}
	return nil
}

func MarshalCommunityGroups(groups []CommunityGroup) (string, error) {
	normalized, err := NormalizeCommunityGroups(groups)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal community groups: %w", err)
	}
	return string(data), nil
}

// ParseCommunityGroups is fail-closed: malformed or manually tampered stored
// settings never reach the user-facing response.
func ParseCommunityGroups(raw string) []CommunityGroup {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []CommunityGroup{}
	}
	var groups []CommunityGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil {
		return []CommunityGroup{}
	}
	normalized, err := NormalizeCommunityGroups(groups)
	if err != nil {
		return []CommunityGroup{}
	}
	return normalized
}

func (s *SettingService) GetCommunityGroups(ctx context.Context) ([]CommunityGroup, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyCommunityGroups)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return []CommunityGroup{}, nil
		}
		return nil, fmt.Errorf("get community groups: %w", err)
	}
	return ParseCommunityGroups(raw), nil
}
