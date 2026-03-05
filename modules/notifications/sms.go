package notifications

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type phoneVerificationCodeRecord struct {
	ID   int
	Code string
}

type smsCodeStorage interface {
	deleteCodesByProfileID(ctx context.Context, profileID int) error
	createCode(ctx context.Context, profileID int, code string) error
	findLatestValidCode(ctx context.Context, profileID int, minCreatedAt time.Time) (*phoneVerificationCodeRecord, error)
	deleteCodeByID(ctx context.Context, id int) error
}

type smsSendFn func(ctx context.Context, phoneNumber, message string) (*sns.PublishOutput, error)

type SMSSender struct {
	store                           smsCodeStorage
	snsClient                       *sns.Client
	senderID                        string
	validationTextExpirationMinutes int
	sendSMS                         smsSendFn
}

func newSMSSender(
	store smsCodeStorage, snsClient *sns.Client, senderID string, validationTextExpirationMinutes int,
) *SMSSender {
	s := &SMSSender{
		store:                           store,
		snsClient:                       snsClient,
		senderID:                        senderID,
		validationTextExpirationMinutes: validationTextExpirationMinutes,
	}
	s.sendSMS = s.sendSMSWithSNS
	return s
}

// CreateConfirmationCode creates and sends a validation code to the given phone number.
func (s *SMSSender) CreateConfirmationCode(
	ctx context.Context, profileID int, phoneNumber string,
) (string, error) {
	if err := s.store.deleteCodesByProfileID(ctx, profileID); err != nil {
		return "", err
	}

	code := generateRandomIntWithNDigits(4)
	codeStr := fmt.Sprintf("%d", code)
	if err := s.store.createCode(ctx, profileID, codeStr); err != nil {
		return "", err
	}

	if _, err := s.sendSMS(ctx, phoneNumber, fmt.Sprintf("Please confirm your phone number for Goship, your code is %d", code)); err != nil {
		return "", err
	}
	return codeStr, nil
}

func (s *SMSSender) VerifyConfirmationCode(ctx context.Context, profileID int, code string) (bool, error) {
	minCreatedAt := time.Now().Add(-time.Minute * time.Duration(s.validationTextExpirationMinutes))
	phoneCodeInDB, err := s.store.findLatestValidCode(ctx, profileID, minCreatedAt)
	if err != nil {
		return false, err
	}
	if code != phoneCodeInDB.Code {
		return false, errors.New("incorrect code")
	}
	if err := s.store.deleteCodeByID(ctx, phoneCodeInDB.ID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SMSSender) sendSMSWithSNS(ctx context.Context, phoneNumber, message string) (*sns.PublishOutput, error) {
	params := &sns.PublishInput{
		Message:     aws.String(message),
		PhoneNumber: aws.String(phoneNumber),
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

// SendSms sends an SMS message to the specified phone number with the given message.
func (s *SMSSender) SendSms(ctx context.Context, phoneNumber, message string) (*sns.PublishOutput, error) {
	return s.sendSMS(ctx, phoneNumber, message)
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

// pow is a simple power function.
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
