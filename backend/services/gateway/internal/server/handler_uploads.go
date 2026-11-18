package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

const maxImageBytes = 5 << 20

var fileNameRE = regexp.MustCompile(`^[a-f0-9]{32}\.(png|jpg|webp)$`)

// sniffImage : le format est déduit du CONTENU (jamais du nom ni du type annoncé). SVG, GIF et PDF sont refusés (scripts, animations).
func sniffImage(b []byte) (ext string, ok bool) {
	switch http.DetectContentType(b) {
	case "image/png":
		return "png", true
	case "image/jpeg":
		return "jpg", true
	case "image/webp":
		return "webp", true
	}
	return "", false
}

// AdminUploadImage enregistre une image du personnel sous un nom dérivé de son contenu (SHA-256) : un même fichier n'est stocké qu'une fois,
// et le nom ne dépend jamais de ce que l'utilisateur a envoyé.
func (h *handler) AdminUploadImage(_ context.Context, req *api.ImageUpload) (api.AdminUploadImageRes, error) {
	data, err := base64.StdEncoding.DecodeString(req.DataBase64)
	if err != nil {
		return nil, apperr.Validation("image illisible", apperr.Field{Pointer: "/dataBase64", Code: apperr.CodeValidation, Message: "contenu base64 invalide"})
	}
	if len(data) == 0 || len(data) > maxImageBytes {
		return nil, apperr.Validation("image trop lourde", apperr.Field{Pointer: "/dataBase64", Code: apperr.CodeValidation, Message: "5 Mo au maximum"})
	}
	ext, ok := sniffImage(data)
	if !ok {
		return nil, apperr.Validation("format non pris en charge", apperr.Field{Pointer: "/dataBase64", Code: apperr.CodeValidation, Message: "PNG, JPEG ou WebP uniquement"})
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:16]) + "." + ext
	dir := h.srv.cfg.UploadDir
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, apperr.Internal(err)
	}
	dst := filepath.Join(dir, name)
	if _, err := os.Stat(dst); errors.Is(err, os.ErrNotExist) {
		tmp, err := os.CreateTemp(dir, ".up-*")
		if err != nil {
			return nil, apperr.Internal(err)
		}
		_, werr := tmp.Write(data)
		cerr := tmp.Close()
		if werr != nil || cerr != nil {
			_ = os.Remove(tmp.Name())
			return nil, apperr.Internal(errors.Join(werr, cerr))
		}
		if err := os.Rename(tmp.Name(), dst); err != nil {
			_ = os.Remove(tmp.Name())
			return nil, apperr.Internal(err)
		}
	}
	return &api.ImageUploaded{URL: "/v1/files/" + name}, nil
}

// GetUploadedImage sert une image téléversée. Le nom est contrôlé par l'expression du contrat ET ici (aucune traversée de dossier possible).
func (h *handler) GetUploadedImage(_ context.Context, p api.GetUploadedImageParams) (api.GetUploadedImageRes, error) {
	if !fileNameRE.MatchString(p.Name) {
		return nil, apperr.NotFound("image introuvable")
	}
	b, err := os.ReadFile(filepath.Join(h.srv.cfg.UploadDir, p.Name)) // ≤ 5 Mo : lue en entier, aucun descripteur laissé ouvert
	if err != nil {
		return nil, apperr.NotFound("image introuvable")
	}
	f := bytes.NewReader(b)
	cache := api.NewOptString("public, max-age=31536000, immutable") // le nom est le hash du contenu : il ne change jamais
	switch filepath.Ext(p.Name) {
	case ".png":
		return &api.GetUploadedImageOKImagePNGHeaders{CacheControl: cache, Response: api.GetUploadedImageOKImagePNG{Data: f}}, nil
	case ".webp":
		return &api.GetUploadedImageOKImageWEBPHeaders{CacheControl: cache, Response: api.GetUploadedImageOKImageWEBP{Data: f}}, nil
	}
	return &api.GetUploadedImageOKImageJpegHeaders{CacheControl: cache, Response: api.GetUploadedImageOKImageJpeg{Data: f}}, nil
}
