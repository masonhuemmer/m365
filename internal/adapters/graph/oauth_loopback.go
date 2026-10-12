package graph

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

func StartLoopback() (string, func(context.Context) (string, error), error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, domain.Auth(err.Error())
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	redirect := "http://localhost:" + port + "/"
	ch := make(chan string, 1)
	srv := &http.Server{ReadHeaderTimeout: 3 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := r.URL.Query().Get("code")
		_, _ = io.WriteString(w, "ok")
		select {
		case ch <- c:
		default:
		}
	})}
	go func() { _ = srv.Serve(ln) }()
	wait := func(ctx context.Context) (string, error) {
		defer func() { _ = srv.Shutdown(context.Background()) }()
		select {
		case <-ctx.Done():
			return "", domain.Auth("login cancelled")
		case c := <-ch:
			if c == "" {
				return "", domain.Auth("login refused")
			}
			return c, nil
		}
	}
	return redirect, wait, nil
}

func LoginPKCE(ctx context.Context, cfg *oauth2.Config, open auth.BrowserOpener, start auth.LoopbackStart) (auth.Blob, error) {
	if open == nil || start == nil {
		return auth.Blob{}, domain.Auth("login not available")
	}
	redirect, wait, err := start()
	if err != nil {
		return auth.Blob{}, err
	}
	cfg.RedirectURL = redirect
	v := oauth2.GenerateVerifier()
	u := cfg.AuthCodeURL("state", oauth2.S256ChallengeOption(v), oauth2.AccessTypeOffline)
	if err := open(u); err != nil {
		return auth.Blob{}, domain.Auth(err.Error())
	}
	code, err := wait(ctx)
	if err != nil {
		return auth.Blob{}, err
	}
	tok, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(v))
	if err != nil {
		return auth.Blob{}, domain.Auth(err.Error())
	}
	scope := ""
	if s, ok := tok.Extra("scope").(string); ok {
		scope = s
	}
	b := auth.Blob{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    tok.Expiry,
		Usable:       tok.AccessToken != "",
		Mail:         strings.Contains(scope, "Mail."),
		Teams:        strings.Contains(scope, "Chat"),
		Calendar:     strings.Contains(scope, "Calendars."),
		Files:        strings.Contains(scope, "Files."),
		Account:      "signed-in",
	}
	if b.AccessToken != "" {
		if acct := fetchMe(ctx, tok.AccessToken); acct != "" {
			b.Account = acct
		}
		if !b.Mail && !b.Teams {
			b.Mail, b.Teams = true, true
		}
	}
	return b, nil
}

func fetchMe(ctx context.Context, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return ""
	}
	var me struct {
		Mail string `json:"mail"`
		UPN  string `json:"userPrincipalName"`
	}
	if err := json.NewDecoder(res.Body).Decode(&me); err != nil {
		return ""
	}
	if me.Mail != "" {
		return me.Mail
	}
	return me.UPN
}

func RealLogin(clientID, tenantID string) auth.LoginFn {
	return func(ctx context.Context) (auth.Blob, error) {
		cfg := PKCEConfig(clientID, tenantID, "")
		cfg.Endpoint.AuthStyle = oauth2.AuthStyleInParams
		return LoginPKCE(ctx, cfg, OpenBrowser, StartLoopback)
	}
}
