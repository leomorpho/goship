package twofa

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

const pendingUserCookieName = "pending_2fa_user"

var ErrPendingUserCookieInvalid = errors.New("pending two factor cookie invalid")

func SetPendingUserCookie(ctx echo.Context, secret string, userID int) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().UTC().Add(5 * time.Minute).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return err
	}
	ctx.SetCookie(&http.Cookie{
		Name:     pendingUserCookieName,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(5 * time.Minute),
		MaxAge:   int((5 * time.Minute).Seconds()),
	})
	return nil
}

func PendingUserIDFromCookie(ctx echo.Context, secret string) (int, error) {
	cookie, err := ctx.Cookie(pendingUserCookieName)
	if err != nil {
		return 0, ErrPendingUserCookieInvalid
	}
	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, ErrPendingUserCookieInvalid
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ErrPendingUserCookieInvalid
	}
	switch value := claims["user_id"].(type) {
	case float64:
		return int(value), nil
	case string:
		userID, err := strconv.Atoi(value)
		if err != nil {
			return 0, ErrPendingUserCookieInvalid
		}
		return userID, nil
	default:
		return 0, ErrPendingUserCookieInvalid
	}
}

func ClearPendingUserCookie(ctx echo.Context) error {
	ctx.SetCookie(&http.Cookie{
		Name:     pendingUserCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Now().UTC().Add(-time.Hour),
	})
	return nil
}
