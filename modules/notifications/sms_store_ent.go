package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/phoneverificationcode"
)

type entSMSCodeStore struct {
	orm *ent.Client
}

// NewSMSSender initializes a new SMSSender with Ent-backed storage and AWS SNS.
func NewSMSSender(orm *ent.Client, region, senderID string, validationTextExpirationMinutes int) (*SMSSender, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}
	client := sns.NewFromConfig(cfg)
	return newSMSSender(
		&entSMSCodeStore{orm: orm},
		client,
		senderID,
		validationTextExpirationMinutes,
	), nil
}

func (s *entSMSCodeStore) deleteCodesByProfileID(ctx context.Context, profileID int) error {
	_, err := s.orm.PhoneVerificationCode.
		Delete().
		Where(phoneverificationcode.ProfileIDEQ(profileID)).
		Exec(ctx)
	return err
}

func (s *entSMSCodeStore) createCode(ctx context.Context, profileID int, code string) error {
	return s.orm.PhoneVerificationCode.
		Create().
		SetCode(code).
		SetProfileID(profileID).
		Exec(ctx)
}

func (s *entSMSCodeStore) findLatestValidCode(
	ctx context.Context, profileID int, minCreatedAt time.Time,
) (*phoneVerificationCodeRecord, error) {
	rec, err := s.orm.PhoneVerificationCode.
		Query().
		Where(
			phoneverificationcode.ProfileIDEQ(profileID),
			phoneverificationcode.CreatedAtGTE(minCreatedAt),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &phoneVerificationCodeRecord{
		ID:   rec.ID,
		Code: rec.Code,
	}, nil
}

func (s *entSMSCodeStore) deleteCodeByID(ctx context.Context, id int) error {
	return s.orm.PhoneVerificationCode.DeleteOneID(id).Exec(ctx)
}
