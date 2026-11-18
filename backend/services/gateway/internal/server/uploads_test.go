package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func upload(r *rig, tok string, data []byte) (int, string) {
	body, _ := json.Marshal(map[string]string{"filename": "../../etc/passwd.png", "dataBase64": base64.StdEncoding.EncodeToString(data)})
	h := map[string]string{"Content-Type": "application/json"}
	if tok != "" {
		h["Authorization"] = "Bearer " + tok
	}
	rec := r.do(req{"POST", "/v1/admin/uploads", string(body), h})
	return rec.Code, rec.Body.String()
}

func TestTeleversementDImages(t *testing.T) {
	dir := t.TempDir()
	r := newRig(t, func(c *server.Config) { c.UploadDir = dir })
	tok := staffToken(t, r, uuid.NewString(), "media:write")

	if code, _ := upload(r, "", pngBytes(t)); code != 401 {
		t.Fatalf("sans jeton : %d", code)
	}
	// Téléverser exige « media:write » : un membre du personnel sans cette permission est refusé.
	if code, _ := upload(r, staffToken(t, r, uuid.NewString(), "content:read"), pngBytes(t)); code != 403 {
		t.Fatalf("sans la permission : %d", code)
	}
	code, out := upload(r, tok, pngBytes(t))
	if code != 201 || !strings.Contains(out, `"url":"/v1/files/`) || !strings.HasSuffix(strings.Trim(strings.TrimSpace(out), `"}`), ".png") {
		t.Fatalf("envoi : %d %s", code, out)
	}
	var res struct{ URL string }
	_ = json.Unmarshal([]byte(out), &res)
	if strings.Contains(res.URL, "passwd") || strings.Contains(res.URL, "..") { // le nom envoyé n'est jamais repris
		t.Fatalf("nom repris de l'utilisateur : %s", res.URL)
	}
	// lecture publique : octets identiques, bon type, cache long
	rec := r.do(req{"GET", res.URL, "", nil})
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(rec.Body.Bytes(), pngBytes(t)) || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("lecture : %d %s %q", rec.Code, rec.Header().Get("Content-Type"), rec.Header().Get("Cache-Control"))
	}
	// idempotent : même contenu, même adresse
	if _, again := upload(r, tok, pngBytes(t)); !strings.Contains(again, res.URL) {
		t.Fatalf("second envoi : %s", again)
	}
	// refusés : SVG (scripts), texte, vide
	for name, data := range map[string][]byte{
		"svg":   []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"texte": []byte("bonjour"),
		"vide":  {},
		"html":  []byte("<html><script>alert(1)</script></html>"),
	} {
		if code, _ := upload(r, tok, data); code != 422 && code != 400 {
			t.Fatalf("%s accepté : %d", name, code)
		}
	}
	// noms invalides ou inexistants
	for _, p := range []string{"/v1/files/..%2F..%2Fetc%2Fpasswd", "/v1/files/zzzz.png", "/v1/files/0123456789abcdef0123456789abcdef.png", "/v1/files/0123456789abcdef0123456789abcdef.svg"} {
		if rec := r.do(req{"GET", p, "", nil}); rec.Code != 404 && rec.Code != 400 {
			t.Fatalf("%s : %d", p, rec.Code)
		}
	}
}
