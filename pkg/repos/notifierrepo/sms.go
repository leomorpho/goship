package notifierrepo

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/phoneverificationcode"
	"github.com/rs/zerolog/log"
)

type SMSSender struct {
	orm                             *ent.Client
	snsClient                       *sns.Client
	senderID                        string
	validationTextExpirationMinutes int
}

// NewSMSSender initializes a new SMSSender with the AWS SNS client
func NewSMSSender(orm *ent.Client, region, senderID string, validationTextExpirationMinutes int) (*SMSSender, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}

	client := sns.NewFromConfig(cfg)
	return &SMSSender{
		orm:       orm,
		snsClient: client,
		senderID:  senderID,
	}, nil
}

func (s *SMSSender) CreateConfirmationCode(
	ctx context.Context, profileID int, phoneNumber string,
) (string, error) {

	_, err := s.orm.PhoneVerificationCode.
		Delete().
		Where(
			phoneverificationcode.ProfileIDEQ(profileID),
		).
		Exec(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}

	code := generateRandomIntWithNDigits(4)

	err = s.orm.PhoneVerificationCode.
		Create().
		SetCode(fmt.Sprintf("%d", code)).
		SetProfileID(profileID).
		Exec(ctx)

	_, err = s.SendSms(ctx, phoneNumber,
		fmt.Sprintf("Please confirm your phone number for Goship, your code is %d", code))
	if err != nil {
		log.Error().Err(err)
		return "", err
	}

	return fmt.Sprintf("%d", code), nil
}

func (s *SMSSender) VerifyConfirmationCode(ctx context.Context, profileID int, code string) (bool, error) {
	phoneCodeInDB, err := s.orm.PhoneVerificationCode.
		Query().
		Where(
			phoneverificationcode.ProfileIDEQ(profileID),
			phoneverificationcode.CreatedAtGTE(time.Now().Add(-time.Minute*time.Duration(s.validationTextExpirationMinutes))),
		).
		Only(ctx)

	if err != nil {
		return false, err
	}

	if code != phoneCodeInDB.Code {
		return false, errors.New("incorrect code")
	}

	err = s.orm.PhoneVerificationCode.
		DeleteOneID(phoneCodeInDB.ID).
		Exec(ctx)

	return true, err
}

// SendSms sends an SMS message to the specified phone number with the given message
func (s *SMSSender) SendSms(ctx context.Context, phoneNumber, message string) (*sns.PublishOutput, error) {
	params := &sns.PublishInput{
		Message:     aws.String(message),
		PhoneNumber: aws.String(phoneNumber), // In international string format
		MessageAttributes: map[string]types.MessageAttributeValue{
			"AWS.SNS.SMS.SenderID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(s.senderID),
			},
			"AWS.SNS.SMS.SMSType": {
				DataType:    aws.String("String"),
				StringValue: aws.String("Transactional"),
			},
		},
	}

	resp, err := s.snsClient.Publish(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}

	return resp, nil
}

// generateRandomIntWithNDigits generates a random integer with n digits.
func generateRandomIntWithNDigits(n int) int {
	if n <= 0 {
		return 0
	}

	min := int(pow(10, n-1))
	max := int(pow(10, n)) - 1

	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

// pow is a simple power function
func pow(base, exp int) int {
	result := 1
	for exp != 0 {
		if exp%2 != 0 {
			result *= base
		}
		exp /= 2
		base *= base
	}
	return result
}
