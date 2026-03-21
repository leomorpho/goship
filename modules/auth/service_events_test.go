package auth

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/events"
	eventtypes "github.com/leomorpho/goship/framework/events/types"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/stretchr/testify/require"
)

func TestServicePublishUserRegistered(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	container := &frameworkbootstrap.Container{
		Web:      e,
		Logger:   e.Logger,
		EventBus: events.NewBus(),
	}
	service := &Service{
		ctr: ui.NewController(container),
	}

	received := make(chan eventtypes.UserRegistered, 1)
	events.Subscribe(container.EventBus, func(_ context.Context, event eventtypes.UserRegistered) error {
		received <- event
		return nil
	})

	before := time.Now().UTC()
	service.publishUserRegistered(ctx, 42, "events@example.com")

	select {
	case event := <-received:
		require.Equal(t, int64(42), event.UserID)
		require.Equal(t, "events@example.com", event.Email)
		require.False(t, event.At.Before(before))
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for user registered event")
	}
}

func TestServicePublishAuthLifecycleEvents(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	container := &frameworkbootstrap.Container{
		Web:      e,
		Logger:   e.Logger,
		EventBus: events.NewBus(),
	}
	service := &Service{
		ctr: ui.NewController(container),
	}

	loggedInCh := make(chan eventtypes.UserLoggedIn, 1)
	loggedOutCh := make(chan eventtypes.UserLoggedOut, 1)
	passwordChangedCh := make(chan eventtypes.PasswordChanged, 1)

	events.Subscribe(container.EventBus, func(_ context.Context, event eventtypes.UserLoggedIn) error {
		loggedInCh <- event
		return nil
	})
	events.Subscribe(container.EventBus, func(_ context.Context, event eventtypes.UserLoggedOut) error {
		loggedOutCh <- event
		return nil
	})
	events.Subscribe(container.EventBus, func(_ context.Context, event eventtypes.PasswordChanged) error {
		passwordChangedCh <- event
		return nil
	})

	service.publishUserLoggedIn(ctx, 7)
	service.publishUserLoggedOut(ctx, 7)
	service.publishPasswordChanged(ctx, 7)

	select {
	case event := <-loggedInCh:
		require.Equal(t, int64(7), event.UserID)
		require.Equal(t, "192.0.2.10", event.IP)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for user logged in event")
	}

	select {
	case event := <-loggedOutCh:
		require.Equal(t, int64(7), event.UserID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for user logged out event")
	}

	select {
	case event := <-passwordChangedCh:
		require.Equal(t, int64(7), event.UserID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for password changed event")
	}
}
