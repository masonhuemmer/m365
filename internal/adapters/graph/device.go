package graph

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/oauth2"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

// deviceLogin runs the OAuth device code flow. The prompt goes to out (stderr in
// production) so stdout stays machine-readable JSON.
func deviceLogin(ctx context.Context, cfg *oauth2.Config, out io.Writer) (auth.Blob, error) {
	da, err := cfg.DeviceAuth(ctx)
	if err != nil {
		return auth.Blob{}, domain.Auth(err.Error())
	}
	fmt.Fprintf(out, "To sign in, open %s and enter the code %s\n", da.VerificationURI, da.UserCode)
	tok, err := cfg.DeviceAccessToken(ctx, da)
	if err != nil {
		if ctx.Err() != nil {
			return auth.Blob{}, domain.Auth("login cancelled")
		}
		return auth.Blob{}, domain.Auth(err.Error())
	}
	return blobFromToken(ctx, tok), nil
}
