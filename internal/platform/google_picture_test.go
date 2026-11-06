package platform

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGooglePictureOnlyAllowsGoogleHTTPSImages(t *testing.T) {
	valid := "https://lh3.googleusercontent.com/a/profile=s96-c"
	if googlePicture(valid) != valid {
		t.Fatal("Google photo rejected")
	}
	for _, raw := range []string{"", "http://lh3.googleusercontent.com/a", "https://lh3.googleusercontent.com.evil.test/a", "https://attacker.test/a", "https://127.0.0.1/a", "data:image/png;base64,aaa", "https://user@lh3.googleusercontent.com/a", "https://lh3.googleusercontent.com:443/a", "https://lh3.googleusercontent.com/a#fragment"} {
		if googlePicture(raw) != "" {
			t.Fatal("untrusted image accepted", raw)
		}
	}
}

func TestGoogleLoginRefreshesAndPersistsPhoto(t *testing.T) {
	p := databasePlatform(t)
	u, _ := signupTest(t, p)
	identity := googleClaims{Email: u.Email, Picture: "https://lh3.googleusercontent.com/a/first=s96-c", RegisteredClaims: jwt.RegisteredClaims{Subject: token("google_")}}
	for _, picture := range []string{identity.Picture, "https://lh3.googleusercontent.com/a/updated=s96-c", "https://attacker.test/photo"} {
		identity.Picture = picture
		linked, err := p.linkGoogle(context.Background(), identity)
		if err != nil {
			t.Fatal(err)
		}
		stored, err := p.user(context.Background(), u.ID)
		if err != nil || stored.AvatarURL != googlePicture(picture) || linked.AvatarURL != stored.AvatarURL || stored.Role != "user" {
			t.Fatal("photo or account changed incorrectly", err)
		}
	}
}
