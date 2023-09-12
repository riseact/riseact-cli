package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"riseact/internal/config"
	"riseact/internal/utils/logger"
	"runtime"
	"time"
)

type OAuthTokenPayload struct {
	ClientId     string `form:"client_id"`
	CodeVerifier string `form:"code_verifier"`
	Code         string `form:"code"`
	RedirectUri  string `form:"redirect_uri"`
	GrantType    string `form:"grant_type"`
}

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresAt    int    `json:"expires_at,omitempty"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

type UserInfo struct {
	Name        string `json:"username"`
	Email       string `json:"email"`
	PartnerID   int    `json:"partner_id"`
	PartnerName string `json:"partner_name"`
}

func generateAuthorizationURL() (string, string) {
	code_verifier, code_challenge := generatePKCEChallenge()

	settings := config.GetAppSettings()

	baseUrl := settings.AccountsHost + "/oauth/authorize"

	values := url.Values{
		"client_id":             {settings.ClientId},
		"redirect_uri":          {settings.RedirectUri},
		"response_type":         {"code"},
		"code_challenge":        {code_challenge},
		"code_challenge_method": {"S256"},
	}

	return fmt.Sprintf("%s?%s", baseUrl, values.Encode()), code_verifier
}

func generateCodeVerifier() (string, error) {
	const length = 43 // Puoi cambiarlo, ma deve essere tra 43 e 128.

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	verifier := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	return verifier, nil
}

func generateCodeChallenge(verifier string) string {
	s256 := sha256.Sum256([]byte(verifier))
	challenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(s256[:])
	return challenge
}

func generatePKCEChallenge() (string, string) {
	code_verifier, err := generateCodeVerifier()

	if err != nil {
		panic(err)
	}

	code_challenge := generateCodeChallenge(code_verifier)
	return code_verifier, code_challenge
}

func launchCallbackWebServer(ch chan *string) {
	stopCh := make(chan struct{})

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		if error, ok := values["error"]; ok {
			fmt.Fprintf(w, "There was and error with your login: %s", error[0])
			ch <- nil
		}

		if code, ok := values["code"]; ok {
			// TODO: serve a nicer web page
			fmt.Fprintf(w, "Authorization retrieved. You can close this window now.")
			ch <- &code[0]
		}

		close(stopCh)
	})

	server := &http.Server{
		Addr:    ":55443",
		Handler: mux,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer server.Shutdown(ctx)
	defer cancel()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	fmt.Println("Awaiting authorization on port 55443...")

	<-stopCh
}

func awaitAuthorizationCode() (string, error) {
	ch := make(chan *string)

	go launchCallbackWebServer(ch)

	code := <-ch

	if code == nil {
		return "", fmt.Errorf("Error while retrivieng authorization code")
	}

	return *code, nil
}

func getAccessToken(auth_code string, code_verifier string) (*AccessToken, error) {
	settings := config.GetAppSettings()
	baseUrl := settings.AccountsHost + "/oauth/token/"

	values := url.Values{
		"client_id":     {settings.ClientId},
		"code_verifier": {code_verifier},
		"code":          {auth_code},
		"redirect_uri":  {settings.RedirectUri},
		"grant_type":    {"authorization_code"},
	}

	resp, err := http.PostForm(baseUrl, values)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var accessToken *AccessToken

	err = json.NewDecoder(resp.Body).Decode(&accessToken)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	accessToken.ExpiresAt = int(time.Now().Unix()) + accessToken.ExpiresIn

	return accessToken, nil
}

func launchBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

func refreshToken() error {
	userSettings, err := config.GetUserSettings()
	settings := config.GetAppSettings()
	baseUrl := settings.AccountsHost + "/oauth/token/"

	values := url.Values{
		"client_id":     {settings.ClientId},
		"refresh_token": {userSettings.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	resp, err := http.PostForm(baseUrl, values)

	if err != nil {
		logger.Debugf("Error while refreshing token: %s", err.Error())
		return fmt.Errorf("Unable to authenticate. Please login again")
	}

	var newAccessToken *AccessToken

	err = json.NewDecoder(resp.Body).Decode(&newAccessToken)

	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	userSettings.AccessToken = newAccessToken.AccessToken
	userSettings.RefreshToken = newAccessToken.RefreshToken

	config.SaveUserSettings(userSettings)

	return nil

}

func getUserInfo(accessToken string) (*UserInfo, error) {
	settings := config.GetAppSettings()
	baseUrl := settings.AccountsHost + "/oauth/userinfo/"

	req, err := http.NewRequest("GET", baseUrl, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	var userInfo *UserInfo

	err = json.NewDecoder(resp.Body).Decode(&userInfo)

	if err != nil {
		return nil, err
	}

	return userInfo, nil
}
