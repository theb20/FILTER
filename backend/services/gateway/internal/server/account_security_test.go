package server_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Dernières requêtes reçues par le faux service identity (vérifie que le compte et la session viennent du JETON).
var (
	lastSessList  *identityv1.ListMySessionsRequest
	lastSessDel   *identityv1.RevokeMySessionRequest
	lastSessOther *identityv1.RevokeMyOtherSessionsRequest
	lastLoginEv   *identityv1.ListMyLoginEventsRequest
	lastImgPut    *identityv1.PutMyImageRequest
	lastImgGet    *identityv1.GetMyImageRequest
	lastSecure    *identityv1.SecureAccountFromAlertRequest
)

func (f *fakeID) ListMySessions(_ context.Context, in *identityv1.ListMySessionsRequest, _ ...grpc.CallOption) (*identityv1.ListMySessionsResponse, error) {
	lastSessList = in
	return &identityv1.ListMySessionsResponse{Sessions: []*identityv1.MySession{
		{Id: uuid.NewString(), Current: true, CreatedAt: timestamppb.Now(), LastSeenAt: timestamppb.Now(), Ip: "41.202.10.1",
			Device: &identityv1.DeviceInfo{Browser: "Chrome", Os: "Windows", Kind: "desktop"}, Location: &identityv1.GeoLocation{City: "Abidjan", CountryCode: "CI", Country: "Côte d'Ivoire", Latitude: 5.36, Longitude: -4.01}},
		{Id: uuid.NewString(), CreatedAt: timestamppb.Now(), LastSeenAt: timestamppb.Now(), Ip: "10.0.0.5", Device: &identityv1.DeviceInfo{Browser: "Navigateur inconnu", Os: "Système inconnu", Kind: "bizarre"}},
	}}, nil
}

func (f *fakeID) RevokeMySession(_ context.Context, in *identityv1.RevokeMySessionRequest, _ ...grpc.CallOption) (*identityv1.RevokeMySessionResponse, error) {
	lastSessDel = in
	return &identityv1.RevokeMySessionResponse{}, nil
}

func (f *fakeID) RevokeMyOtherSessions(_ context.Context, in *identityv1.RevokeMyOtherSessionsRequest, _ ...grpc.CallOption) (*identityv1.RevokeMyOtherSessionsResponse, error) {
	lastSessOther = in
	return &identityv1.RevokeMyOtherSessionsResponse{Revoked: 2}, nil
}

func (f *fakeID) ListMyLoginEvents(_ context.Context, in *identityv1.ListMyLoginEventsRequest, _ ...grpc.CallOption) (*identityv1.ListMyLoginEventsResponse, error) {
	lastLoginEv = in
	return &identityv1.ListMyLoginEventsResponse{Events: []*identityv1.LoginEvent{{Id: uuid.NewString(), At: timestamppb.Now(), Ip: "41.202.10.1", Method: "google", NewDevice: true,
		Device: &identityv1.DeviceInfo{Browser: "Firefox", Os: "Android", Kind: "mobile"}}}}, nil
}

func (f *fakeID) SecureAccountFromAlert(_ context.Context, in *identityv1.SecureAccountFromAlertRequest, _ ...grpc.CallOption) (*identityv1.SecureAccountFromAlertResponse, error) {
	lastSecure = in
	return &identityv1.SecureAccountFromAlertResponse{}, nil
}

func (f *fakeID) PutMyImage(_ context.Context, in *identityv1.PutMyImageRequest, _ ...grpc.CallOption) (*identityv1.PutMyImageResponse, error) {
	lastImgPut = in
	return &identityv1.PutMyImageResponse{ContentType: "image/png", UpdatedAt: timestamppb.Now()}, nil
}

func (f *fakeID) GetMyImage(_ context.Context, in *identityv1.GetMyImageRequest, _ ...grpc.CallOption) (*identityv1.GetMyImageResponse, error) {
	lastImgGet = in
	return &identityv1.GetMyImageResponse{Data: []byte("\x89PNG..."), ContentType: "image/png", UpdatedAt: timestamppb.Now()}, nil
}

func (f *fakeID) DeleteMyImage(_ context.Context, _ *identityv1.DeleteMyImageRequest, _ ...grpc.CallOption) (*identityv1.DeleteMyImageResponse, error) {
	return &identityv1.DeleteMyImageResponse{}, nil
}

func TestSecuriteDuCompteExigeLaConnexionClient(t *testing.T) {
	r := newRig(t, nil)
	routes := []req{
		{"GET", "/v1/me/sessions", "", nil},
		{"DELETE", "/v1/me/sessions/" + uuid.NewString(), "", nil},
		{"POST", "/v1/me/sessions/revoke-others", "", nil},
		{"GET", "/v1/me/login-activity", "", nil},
		{"GET", "/v1/me/images/avatar", "", nil},
		{"PUT", "/v1/me/images/cover", `{"data":"aGVsbG8="}`, nil},
		{"DELETE", "/v1/me/images/avatar", "", nil},
	}
	staff := bearer(staffToken(t, r, uuid.NewString()))
	for _, q := range routes {
		if rec := r.do(q); rec.Code != 401 {
			t.Errorf("%s %s sans jeton : %d", q.method, q.path, rec.Code)
		}
		q.headers = staff
		if rec := r.do(q); rec.Code != 401 {
			t.Errorf("%s %s avec un jeton du personnel : %d", q.method, q.path, rec.Code)
		}
	}
}

func TestSecuriteDuCompteLeCompteVientDuJeton(t *testing.T) {
	r := newRig(t, nil)
	sub := uuid.NewString()
	tok := bearer(customerTokenFor(t, r, sub)) // sid = "sid-c"

	rec := r.do(req{"GET", "/v1/me/sessions", "", tok})
	if rec.Code != 200 || lastSessList.GetUserId() != sub || lastSessList.GetCurrentSessionId() != "sid-c" {
		t.Fatalf("liste : %d %+v %s", rec.Code, lastSessList, rec.Body.String())
	}
	var list struct {
		Items []struct {
			Current  bool            `json:"current"`
			Device   map[string]any  `json:"device"`
			Location *map[string]any `json:"location"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Items) != 2 || !list.Items[0].Current || list.Items[0].Location == nil || list.Items[1].Location != nil || list.Items[1].Device["kind"] != "unknown" {
		t.Fatalf("réponse : %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"location":null`) {
		t.Fatalf("une position inconnue doit valoir null : %s", rec.Body.String())
	}

	id := uuid.NewString()
	if rec := r.do(req{"DELETE", "/v1/me/sessions/" + id, "", tok}); rec.Code != 204 || lastSessDel.GetUserId() != sub || lastSessDel.GetSessionId() != id {
		t.Fatalf("déconnexion : %d %+v", rec.Code, lastSessDel)
	}
	if rec := r.do(req{"POST", "/v1/me/sessions/revoke-others", "", tok}); rec.Code != 200 || lastSessOther.GetCurrentSessionId() != "sid-c" || !strings.Contains(rec.Body.String(), `"revoked":2`) {
		t.Fatalf("autres appareils : %d %+v %s", rec.Code, lastSessOther, rec.Body.String())
	}
	if rec := r.do(req{"GET", "/v1/me/login-activity?page=2&pageSize=5", "", tok}); rec.Code != 200 || lastLoginEv.GetUserId() != sub || lastLoginEv.GetPage().GetPage() != 2 || lastLoginEv.GetPage().GetPageSize() != 5 || !strings.Contains(rec.Body.String(), `"method":"google"`) {
		t.Fatalf("historique : %d %+v %s", rec.Code, lastLoginEv, rec.Body.String())
	}

	data := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\n...."))
	if rec := r.do(req{"PUT", "/v1/me/images/avatar", `{"data":"` + data + `"}`, tok}); rec.Code != 200 || lastImgPut.GetUserId() != sub || lastImgPut.GetKind() != "avatar" || !strings.HasPrefix(string(lastImgPut.GetData()), "\x89PNG") {
		t.Fatalf("envoi : %d %+v", rec.Code, lastImgPut)
	}
	if rec := r.do(req{"GET", "/v1/me/images/cover", "", tok}); rec.Code != 200 || lastImgGet.GetUserId() != sub || lastImgGet.GetKind() != "cover" {
		t.Fatalf("lecture : %d %+v", rec.Code, lastImgGet)
	}
	if rec := r.do(req{"PUT", "/v1/me/images/banner", `{"data":"` + data + `"}`, tok}); rec.Code != 400 {
		t.Fatalf("type d'image inconnu : %d", rec.Code)
	}
	// La taille exacte (600 Kio) est vérifiée par identity ; le gateway borne le corps à 1 Mio.
	big := base64.StdEncoding.EncodeToString(make([]byte, 900*1024))
	if rec := r.do(req{"PUT", "/v1/me/images/avatar", `{"data":"` + big + `"}`, tok}); rec.Code != 413 && rec.Code != 400 {
		t.Fatalf("corps trop lourd : %d", rec.Code)
	}
}

func TestLienCeNEtaitPasMoiSansConnexion(t *testing.T) {
	r := newRig(t, nil)
	if rec := r.do(req{"POST", "/v1/auth/login-alert/secure", `{"token":"0123456789abcdef0123"}`, nil}); rec.Code != 204 || lastSecure.GetToken() != "0123456789abcdef0123" {
		t.Fatalf("lien de secours : %d %+v", rec.Code, lastSecure)
	}
	if rec := r.do(req{"POST", "/v1/auth/login-alert/secure", `{"token":"court"}`, nil}); rec.Code != 422 {
		t.Fatalf("jeton mal formé : %d", rec.Code)
	}
}
