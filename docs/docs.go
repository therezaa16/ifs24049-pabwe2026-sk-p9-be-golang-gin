// Package docs menyajikan spesifikasi OpenAPI 2.0 (Swagger) Delcom Blog.
// Spesifikasi disusun secara programatik agar tetap sinkron dengan rute.
package docs

import (
	"encoding/json"

	"github.com/swaggo/swag"
)

type op struct {
	method, path, summary, tag string
	auth                       bool
	body                       string
}

var operations = []op{
	{"get", "/public/categories", "Daftar kategori artikel", "Public", false, ""},
	{"get", "/public/posts", "Daftar artikel terbit (?q=&category=)", "Public", false, ""},
	{"get", "/public/posts/{slug}", "Detail artikel terbit berdasarkan slug", "Public", false, ""},
	{"post", "/auth/register", "Registrasi akun penulis", "Auth", false, "RegisterRequest"},
	{"post", "/auth/login", "Login & penerbitan JWT", "Auth", false, "LoginRequest"},
	{"post", "/auth/logout", "Logout", "Auth", true, ""},
	{"get", "/auth/profile", "Ambil profil", "Auth", true, ""},
	{"put", "/auth/profile", "Ubah email / password", "Auth", true, "UpdateProfileRequest"},
	{"get", "/posts", "Daftar artikel milik penulis", "Posts", true, ""},
	{"post", "/posts", "Buat artikel", "Posts", true, "CreatePostRequest"},
	{"get", "/posts/stats", "Statistik 3 kartu penulis", "Posts", true, ""},
	{"post", "/posts/ai-generate", "Generate draf artikel dengan AI + Search Engine", "Posts", true, "GenerateRequest"},
	{"get", "/posts/{id}", "Detail artikel milik penulis", "Posts", true, ""},
	{"put", "/posts/{id}", "Perbarui artikel", "Posts", true, "UpdatePostRequest"},
	{"patch", "/posts/{id}/toggle-publish", "Toggle Draf/Terbit", "Posts", true, ""},
	{"delete", "/posts/{id}", "Hapus artikel", "Posts", true, ""},
	{"get", "/admin/users", "Daftar pengguna + statistik artikel", "Admin", true, ""},
	{"get", "/admin/posts", "Seluruh artikel semua penulis", "Admin", true, ""},
	{"get", "/admin/users/{id}/posts", "Artikel milik pengguna tertentu", "Admin", true, ""},
}

func obj(props map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": props}
}

func str() map[string]any { return map[string]any{"type": "string"} }

func definitions() map[string]any {
	return map[string]any{
		"RegisterRequest": obj(map[string]any{
			"username": str(), "email": str(), "password": str(),
			"role": map[string]any{"type": "string", "enum": []string{"admin", "member"}},
		}),
		"LoginRequest":         obj(map[string]any{"username": str(), "password": str()}),
		"UpdateProfileRequest": obj(map[string]any{"email": str(), "password": str()}),
		"CreatePostRequest": obj(map[string]any{
			"title": str(), "category": str(), "excerpt": str(), "content": str(),
			"cover_image_path": str(), "is_published": map[string]any{"type": "boolean"},
		}),
		"UpdatePostRequest": obj(map[string]any{
			"title": str(), "category": str(), "excerpt": str(), "content": str(),
			"cover_image_path": str(), "is_published": map[string]any{"type": "boolean"},
		}),
		"GenerateRequest": obj(map[string]any{
			"topic": str(), "category": str(), "use_search": map[string]any{"type": "boolean"},
		}),
	}
}

// JSON menghasilkan dokumen Swagger; host/scheme kosong berarti dibiarkan relatif.
func JSON(host, scheme string) []byte {
	paths := map[string]map[string]any{}
	for _, o := range operations {
		params := []map[string]any{}
		if o.body != "" {
			params = append(params, map[string]any{
				"in": "body", "name": "body", "required": true,
				"schema": map[string]any{"$ref": "#/definitions/" + o.body},
			})
		}
		for _, name := range []string{"slug", "id"} {
			if contains(o.path, "{"+name+"}") {
				params = append(params, map[string]any{"in": "path", "name": name, "required": true, "type": "string"})
			}
		}
		detail := map[string]any{
			"summary": o.summary, "tags": []string{o.tag}, "parameters": params,
			"produces":  []string{"application/json"},
			"consumes":  []string{"application/json"},
			"responses": map[string]any{"200": map[string]any{"description": "OK"}},
		}
		if o.auth {
			detail["security"] = []map[string][]string{{"BearerAuth": {}}}
		}
		if paths[o.path] == nil {
			paths[o.path] = map[string]any{}
		}
		paths[o.path][o.method] = detail
	}
	doc := map[string]any{
		"swagger": "2.0",
		"info": map[string]any{
			"title": "Delcom Blog API", "version": "1.0",
			"description": "REST API blog multi-tier dengan AI generator terintegrasi Search Engine.",
		},
		"basePath": "/api",
		"paths":    paths,
		"securityDefinitions": map[string]any{
			"BearerAuth": map[string]any{
				"type": "apiKey", "name": "Authorization", "in": "header",
				"description": "Format: Bearer <token>",
			},
		},
		"definitions": definitions(),
	}
	if host != "" {
		doc["host"] = host
	}
	if scheme != "" {
		doc["schemes"] = []string{scheme}
	}
	b, _ := json.MarshalIndent(doc, "", "  ")
	return b
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

type spec struct{}

// ReadDoc memenuhi antarmuka swag.Swagger untuk Swagger UI (/swagger/doc.json).
func (spec) ReadDoc() string { return string(JSON("", "")) }

func init() { swag.Register(swag.Name, spec{}) }
