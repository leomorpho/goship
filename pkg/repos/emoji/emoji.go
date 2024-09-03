package emojirepo

import (
	"context"
	"fmt"
	"strings"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/emojis"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

type EmojiRepo struct {
	orm *ent.Client
}

func NewEmojiRepo(orm *ent.Client) *EmojiRepo {
	return &EmojiRepo{
		orm: orm,
	}
}

func (e *EmojiRepo) GetOrCreate(
	ctx context.Context, emoji domain.Emoji,
) (*domain.Emoji, error) {
	// Try to find an existing emoji by its unified_code.
	entEmoji, err := e.orm.Emojis.
		Query().
		Where(emojis.UnifiedCodeEQ(emoji.UnifiedCode)).
		Only(ctx)

	if err != nil && !ent.IsNotFound(err) {
		// An error occurred that isn't due to the emoji not being found.
		return nil, err
	}

	if ent.IsNotFound(err) {
		// Emoji not found, create a new one.
		entEmoji, err = e.orm.Emojis.
			Create().
			SetUnifiedCode(emoji.UnifiedCode).
			SetShortcode(emoji.ShortCode).
			Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	// No update needed, return the existing record.
	return &domain.Emoji{
		ID:          entEmoji.ID,
		UnifiedCode: entEmoji.UnifiedCode,
		ShortCode:   entEmoji.Shortcode,
	}, nil
}

// GetRootEmojiFromShortcode takes an emoji shortcode with potential skin tone modifiers and returns the root emoji shortcode.
func GetRootEmojiFromShortcode(shortcode string) string {
	// Shortcodes for emojis with skin tone modifiers are typically formatted as :emoji::skin-tone-X:
	// Split the shortcode at colons to isolate the root emoji part
	parts := strings.Split(shortcode, ":")
	if len(parts) > 1 {
		return fmt.Sprintf(":%s:", parts[1]) // Reconstruct the root shortcode with colons
	}
	return shortcode // Return the original shortcode if no modifiers are found
}

// GetRootEmojiFromUnifiedCode takes a unified emoji code with potential skin tone or other modifiers and returns the root emoji code.
func GetRootEmojiFromUnifiedCode(unifiedCode string) string {
	// Unified codes for emojis with modifiers are typically formatted as XXXXXX-YYYYY
	// Split the code at the hyphen to isolate the root emoji part
	parts := strings.Split(unifiedCode, "-")
	return parts[0] // Return the root part of the unified code
}
