package emailsmanager

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"

	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/emailsubscription"
	"github.com/mikestefanello/pagoda/ent/emailsubscriptiontype"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/rs/zerolog/log"
)

type ErrAlreadySubscribed struct {
	EmailList string
	Err       error
}

func (e *ErrAlreadySubscribed) Error() string {
	return fmt.Sprintf("email address is already subscribed to email list %s, error is: %v", e.EmailList, e.Err)
}

var ErrInvalidEmailConfirmationCode = errors.New("email confirmation code is invalid")
var ErrEmailSyntaxInvalid = errors.New("email address syntax is invalid")
var ErrEmailAddressInvalidCatchAll = errors.New("invalid email address")

type ErrEmailVerificationFailed struct {
	Err error
}

// Error implements the error interface for CustomError
func (e *ErrEmailVerificationFailed) Error() string {
	return fmt.Sprintf("verify email address failed, error is: %v", e.Err)
}

type EmailSubscriptionRepo struct {
	orm           *ent.Client
	emailVerifier *emailverifier.Verifier
}

func NewEmailSubscriptionRepo(orm *ent.Client) *EmailSubscriptionRepo {
	return &EmailSubscriptionRepo{
		orm:           orm,
		emailVerifier: emailverifier.NewVerifier(),
	}
}

// VerifyEmailValidity confirms an email is valid according to our rules
func (er *EmailSubscriptionRepo) VerifyEmailValidity(email string) error {
	ret, err := er.emailVerifier.Verify(email)
	if err != nil {
		return &ErrEmailVerificationFailed{Err: err}
	}
	if !ret.Syntax.Valid {
		return ErrEmailSyntaxInvalid
	}
	return nil
}

// CreateNewSubscriptionList creates a new subscription list, which people can subscribe to individually
func (er *EmailSubscriptionRepo) CreateNewSubscriptionList(
	ctx context.Context, emailList domain.EmailSubscriptionList,
) error {
	_, err := er.orm.EmailSubscriptionType.
		Create().SetActive(true).SetName(emailsubscriptiontype.Name(emailList.Value)).Save(ctx)

	// Check for a unique constraint violation error
	if ent.IsConstraintError(err) {
		// This error is because the email list already exists; return nil or a custom error if needed
		return nil
	}
	return err
}

// SSESubscribe subscribes an email to a mailing list.
func (er *EmailSubscriptionRepo) SSESubscribe(
	ctx context.Context, email string, emailList domain.EmailSubscriptionList, latitude, longitude *float64,
) (*domain.EmailSubscription, error) {

	if (latitude != nil && longitude == nil) || (latitude == nil && longitude != nil) {
		log.Fatal().Str("error", "both latitude/longitude should either be nil or have a value")
	}
	if err := er.VerifyEmailValidity(email); err != nil {
		log.Error().Err(err)
		return nil, ErrEmailAddressInvalidCatchAll
	}

	// Check if email is already subscribed
	alreadySubscribed, err := er.orm.EmailSubscription.
		Query().
		Where(
			emailsubscription.EmailEQ(email),
		).
		QuerySubscriptions().
		Where(emailsubscriptiontype.NameEQ(emailsubscriptiontype.Name(emailList.Value))).
		Exist(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if alreadySubscribed {
		return nil, &ErrAlreadySubscribed{Err: err, EmailList: emailList.Value}
	}

	subscriptionEmailListObj, err := er.orm.EmailSubscriptionType.
		Query().
		Where(
			emailsubscriptiontype.Active(true),
			emailsubscriptiontype.NameEQ(emailsubscriptiontype.Name(emailList.Value)),
		).Only(ctx)
	if err != nil {
		return nil, err
	}

	var confirmationCode string
	existingSubscription, err := er.orm.EmailSubscription.
		Query().
		Where(
			emailsubscription.EmailEQ(email),
		).Only(ctx)
	if ent.IsNotFound(err) {
		confirmationCode, err = generateUniqueCode()
		if err != nil {
			return nil, err
		}
		createQuery := er.orm.EmailSubscription.
			Create().
			SetEmail(email).
			SetConfirmationCode(confirmationCode).
			AddSubscriptions(subscriptionEmailListObj)

		if latitude != nil && longitude != nil {
			createQuery.SetLatitude(*latitude).SetLongitude(*longitude)
		}
		existingSubscription, err = createQuery.Save(ctx)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	updateQuery := er.orm.EmailSubscription.
		UpdateOne(existingSubscription).
		AddSubscriptions(subscriptionEmailListObj)

	if latitude != nil && longitude != nil {
		updateQuery.SetLatitude(*latitude).SetLongitude(*longitude)
	}

	_, err = updateQuery.Save(ctx)

	return &domain.EmailSubscription{
		ID:               existingSubscription.ID,
		Email:            existingSubscription.Email,
		Verified:         existingSubscription.Verified,
		ConfirmationCode: existingSubscription.ConfirmationCode,
		Lat:              existingSubscription.Latitude,
		Lon:              existingSubscription.Longitude,
	}, err
}

func (er *EmailSubscriptionRepo) SSEUnsubscribe(ctx context.Context, email string, token string, emailList domain.EmailSubscriptionList) error {
	// Retrieve the subscription using the email and token for verification
	getSubscriptionQuery := er.orm.EmailSubscription.
		Query().
		Where(
			emailsubscription.EmailEQ(email),
		).
		WithSubscriptions()
	subscription, err := getSubscriptionQuery.Only(ctx)
	if err != nil {
		return err
	}

	subscriptionEmailListObj, err := er.orm.EmailSubscriptionType.
		Query().
		Where(
			emailsubscriptiontype.NameEQ(emailsubscriptiontype.Name(emailList.Value)),
		).Only(ctx)
	if err != nil {
		return err
	}

	// Remove the subscription
	_, err = subscription.
		Update().RemoveSubscriptions(subscriptionEmailListObj).
		Save(ctx)
	if err != nil {
		return nil
	}

	// Refresh the object from DB with its subscription edge
	subscription, err = getSubscriptionQuery.Only(ctx)
	if err != nil {
		return err
	}

	// If subscriber is not subscribed to any more list, delete them.
	if len(subscription.Edges.Subscriptions) == 0 {
		err = er.orm.EmailSubscription.DeleteOne(subscription).Exec(ctx)
		if err != nil {
			return err
		}
	} else {
		confirmationCode, err := generateUniqueCode()
		if err != nil {
			return err
		}
		_, err = subscription.Update().SetConfirmationCode(confirmationCode).Save(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (er *EmailSubscriptionRepo) ConfirmSubscription(ctx context.Context, code string) error {
	// Find subscription by email and code
	subscription, err := er.orm.EmailSubscription.
		Query().
		Where(emailsubscription.ConfirmationCodeEQ(code)).
		Only(ctx)

	if err != nil {
		return ErrInvalidEmailConfirmationCode
	}
	if subscription.Verified {
		return nil
	}

	confirmationCode, err := generateUniqueCode()
	if err != nil {
		return err
	}

	// Update subscription status and confirmation code
	_, err = subscription.Update().SetVerified(true).SetConfirmationCode(confirmationCode).Save(ctx)
	return err
}

func generateUniqueCode() (string, error) {
	// Define the size of the token
	const tokenSize = 32 // 32 bytes will give a sufficiently long token

	// Create a byte slice to hold the random bytes
	tokenBytes := make([]byte, tokenSize)

	// Generate random bytes
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	// Encode the bytes to a base64 string
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// generateInvitationCode generates a unique code of a specified length containing only letters and numbers.
func generateInvitationCode(length int) (string, error) {
	// Currently using 62 chars in the usable set.
	// For 10-char code: 839,299,365,868,340,200 possibilities
	// For 8-char code: 218,340,105,584,896 possibilities
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}

	return string(code), nil
}
