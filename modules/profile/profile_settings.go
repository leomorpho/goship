package profiles

import (
	"context"
	"time"

	dbgen "github.com/leomorpho/goship/db/gen"
)

// ProfileSettings contains the profile fields used by settings/controllers.
type ProfileSettings struct {
	ID              int
	Bio             string
	Birthdate       time.Time
	CountryCode     string
	PhoneNumberE164 string
	PhoneVerified   bool
	FullyOnboarded  bool
}

func (p *ProfileService) GetProfileSettingsByID(ctx context.Context, profileID int) (*ProfileSettings, error) {
	if p.db == nil {
		return nil, ErrProfileDBNotConfigured
	}
	prof, err := dbgen.GetProfileSettingsByID(ctx, p.db, p.dbDialect, profileID)
	if err != nil {
		return nil, err
	}
	countryCode := ""
	if prof.CountryCode.Valid {
		countryCode = prof.CountryCode.String
	}
	phoneE164 := ""
	if prof.PhoneNumberE164.Valid {
		phoneE164 = prof.PhoneNumberE164.String
	}
	birthdate := time.Time{}
	if prof.Birthdate.Valid {
		birthdate = prof.Birthdate.Time
	}
	return &ProfileSettings{
		ID:              prof.ID,
		Bio:             prof.Bio,
		Birthdate:       birthdate,
		CountryCode:     countryCode,
		PhoneNumberE164: phoneE164,
		PhoneVerified:   prof.PhoneVerified,
		FullyOnboarded:  prof.FullyOnboarded,
	}, nil
}

func (p *ProfileService) UpdateProfileBio(ctx context.Context, profileID int, bio string) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	return dbgen.UpdateProfileBioByID(ctx, p.db, p.dbDialect, profileID, bio)
}

func (p *ProfileService) UpdateProfilePhone(ctx context.Context, profileID int, countryCode, phoneE164 string) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	return dbgen.UpdateProfilePhoneByID(ctx, p.db, p.dbDialect, profileID, countryCode, phoneE164)
}

func (p *ProfileService) MarkProfileFullyOnboarded(ctx context.Context, profileID int) error {
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}
	return dbgen.MarkProfileFullyOnboardedByID(ctx, p.db, p.dbDialect, profileID)
}
